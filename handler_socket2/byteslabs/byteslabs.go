package byteslabs

import (
	"sync"
	"sync/atomic"
)

const mem_chunks_count = 8
const slab_size = 40000
const slab_count = 100

type mem_chunk struct {
	memory []byte

	slab_used [slab_count]bool
	mu        sync.Mutex

	stat_allocated        int64
	stat_alloc_full       int
	stat_alloc_full_small int
	stat_alloc_tail       int
	stat_oom              int
	stat_routed           int // slab allocation was routed to another SLAB
	stat_routed_alloc     int // allocations in routed SLAB

	used_slab_count int32
}

type Allocator struct {
	mem_chunk *mem_chunk

	slab_used       []int // array of all slabs used
	slab_free_space []int // free space in each slab used

	is_additional  bool
	addl_allocator *Allocator
}

var mem_chunks [mem_chunks_count]*mem_chunk

func init() {
	for k, _ := range mem_chunks {
		mem_chunks[k] = &mem_chunk{memory: make([]byte, slab_size*slab_count)}
	}
}

var curr_mem_chunk = uint32(0)

func MakeAllocator() *Allocator {

	_mc := atomic.AddUint32(&curr_mem_chunk, 1)
	_mc = _mc % mem_chunks_count

	t1 := make([]int, 0, 5)
	t2 := make([]int, 0, 5)
	return &Allocator{mem_chunk: mem_chunks[_mc], slab_used: t1, slab_free_space: t2}
}

// NOTE: This will always use locking facilities provided by Allocate function!
func (this *Allocator) take_additional(slab_needed int) {
	// we won't create additional allocator if the allocator is additional already
	if this.is_additional || this.addl_allocator != nil {
		return
	}

	best_slab := -1
	chunks_free := -1
	for i := 0; i < mem_chunks_count; i++ {
		if mem_chunks[i] == this.mem_chunk {
			continue
		}

		_free := slab_count - int(atomic.LoadInt32(&mem_chunks[i].used_slab_count))
		if _free >= slab_needed && _free > chunks_free {
			best_slab = i
			chunks_free = _free
		}
	}
	if best_slab == -1 {
		return
	}

	t1 := make([]int, 0, 5)
	t2 := make([]int, 0, 5)
	this.addl_allocator = &Allocator{mem_chunk: mem_chunks[best_slab], is_additional: true,
		slab_used: t1, slab_free_space: t2}
}

func (this *Allocator) Release() {

	if len(this.slab_used) == 0 && this.addl_allocator == nil {
		return
	}

	mem_chunk := this.mem_chunk

	mem_chunk.mu.Lock()
	curr_used := len(this.slab_used)
	for _, v := range this.slab_used {
		mem_chunk.slab_used[v] = false
	}
	this.slab_free_space = this.slab_free_space[:0]
	this.slab_used = this.slab_used[:0]
	atomic.AddInt32(&mem_chunk.used_slab_count, int32(-curr_used))

	_allocator_to_clean := this.addl_allocator
	this.addl_allocator = nil
	mem_chunk.mu.Unlock()

	if _allocator_to_clean != nil {
		_allocator_to_clean.Release()
	}

}

func (this *Allocator) _alloc(mc *mem_chunk, slab_num, slabs_needed, slab_free, size int) []byte {

	start_pos := slab_num * slab_size
	//fmt.Printf("ALLOC FULLChunk %d size %d [ %d - %d ]\n", slab_num, size, start_pos, start_pos+size)

	for slabs_needed > 0 {
		this.slab_used = append(this.slab_used, slab_num)

		// last slab - add free space here!
		if slabs_needed == 1 {
			this.slab_free_space = append(this.slab_free_space, slab_free)
		} else {
			this.slab_free_space = append(this.slab_free_space, 0)
		}

		mc.slab_used[slab_num] = true

		slabs_needed--
		slab_num++
	}

	return mc.memory[start_pos : start_pos : start_pos+size]
}

func (this *Allocator) Allocate(size int) []byte {

	if size <= 96 {
		return make([]byte, 0, size)
	}

	this.mem_chunk.mu.Lock()
	slb_mem := this.allocate_slab(size)

	_addl := this.addl_allocator
	if slb_mem == nil && _addl == nil && this.is_additional == false {
		this.take_additional((size / slab_size) + 5)
		_addl = this.addl_allocator
	}
	if slb_mem == nil && _addl != nil {
		this.mem_chunk.stat_routed++
	}
	this.mem_chunk.mu.Unlock()

	if slb_mem == nil && _addl != nil {
		this.addl_allocator.mem_chunk.mu.Lock()
		slb_mem = this.addl_allocator.allocate_slab(size)
		this.addl_allocator.mem_chunk.mu.Unlock()
	}

	if slb_mem == nil {
		slb_mem = make([]byte, 0, size)
	}
	return slb_mem
}

func (this *Allocator) allocate_slab(size int) []byte {

	slab_free := (slab_size - (size % slab_size)) % slab_size
	slabs_needed := size / slab_size
	if slab_free > 0 {
		slabs_needed++
	}
	mem_chunk := this.mem_chunk

	if slabs_needed > slab_count-int(mem_chunk.used_slab_count) {
		mem_chunk.stat_oom++
		return nil
	}

	// maybe we can put some data into slabs already allocated by us!
	if slabs_needed <= 1 {
		min_space, min_key := -1, -1
		for k, v := range this.slab_free_space {
			if v >= size && (min_space == -1 || v < min_space) {
				min_key = k
				min_space = v
			}
		}

		if min_key > -1 {
			this.slab_free_space[min_key] -= size
			slab_num := this.slab_used[min_key]
			start_pos := slab_num*slab_size + (slab_size - min_space)

			//fmt.Printf("ALLOC ++chunk %d size %d [ %d - %d ]\n", min_key, size, start_pos, start_pos+size)
			mem_chunk.stat_alloc_tail++
			if this.is_additional {
				mem_chunk.stat_routed_alloc++
			}
			return mem_chunk.memory[start_pos : start_pos : start_pos+size]
		}
	}

	// allocate single slab at the end, if 1 is enough space
	if slabs_needed <= 1 {
		pos := len(mem_chunk.slab_used) - 1
		for ; pos >= 0; pos-- {
			if !mem_chunk.slab_used[pos] {
				break
			}
		}
		if pos > -1 {
			mem_chunk.stat_alloc_full_small++
			if this.is_additional {
				mem_chunk.stat_routed_alloc++
			}

			atomic.AddInt32(&this.mem_chunk.used_slab_count, int32(slabs_needed))
			return this._alloc(mem_chunk, pos, slabs_needed, slab_free, size)
		}
	}

	if slabs_needed > 1 {
		free_slabs, pos := 0, 0
		for pos = 0; pos < len(mem_chunk.slab_used); pos++ {
			if !mem_chunk.slab_used[pos] {
				free_slabs++

				if free_slabs == slabs_needed {
					break
				}
			} else {
				free_slabs = 0
			}
		}

		if free_slabs == slabs_needed {
			mem_chunk.stat_alloc_full++
			if this.is_additional {
				mem_chunk.stat_routed_alloc++
			}
			atomic.AddInt32(&this.mem_chunk.used_slab_count, int32(slabs_needed))
			return this._alloc(mem_chunk, pos-slabs_needed+1, slabs_needed, slab_free, size)
		}
	}

	mem_chunk.stat_oom++
	return nil
}
