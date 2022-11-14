package byteslabs

import (
	"fmt"
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

type BSManager struct {
	mem_chunks_count int // number of memory chunks
	slab_size        int // size of single memory slab
	slab_count       int // number of slabs in a chunk
	mem_chunks       []*mem_chunk
}

func Make(mem_chunks_count, slab_size, slab_count int) *BSManager {
	_mc := make([]*mem_chunk, mem_chunks_count)
	for i := 0; i < len(_mc); i++ {
		_mc[i] = &mem_chunk{memory: make([]byte, slab_size*slab_count)}
	}
	return &BSManager{mem_chunks_count, slab_size, slab_count, _mc}
}

func init() {
	for k, _ := range mem_chunks {
		mem_chunks[k] = &mem_chunk{memory: make([]byte, slab_size*slab_count)}
	}
}

/*
func AA() {

	a := MakeAllocator()
	b := a.Allocate(2)
	c := a.Allocate(2)
	a.Allocate(2)
	a.Allocate(2)
	b[0] = 1
	b[1] = 2
	c[0] = 3
	c[1] = 4

	fmt.Println("U", b)
	fmt.Println("U", c)

	a.Allocate(1000)
	a.Allocate(50)
	a.Allocate(50)
	a.Allocate(8900)
	a.Allocate(18900)
	a.Allocate(18900)
	a.Allocate(10)
	a.Allocate(4440)
	a.Allocate(440)
	a.Release()

	var wg sync.WaitGroup

	ts := NewTimeSpan()

	for ii := 0; ii < 20; ii++ {
		wg.Add(1)
		go func() {
			a := MakeAllocator()
			for i := 0; i < 20000; i++ {
				a.Allocate(1400)
				if i%4 == 0 {
					a.Release()
				}
			}
			a.Release()
			wg.Done()
		}()
	}
	wg.Wait()

	fmt.Println("TOOK SLAB", ts.Get())

	ts = NewTimeSpan()

	for ii := 0; ii < 20; ii++ {
		wg.Add(1)
		func() {
			pool := bytepool.MakeBytePool()
			for i := 0; i < 20000; i++ {
				pool.GetBytePool(1400)
				if i%4 == 0 {
					pool.Release()
				}
			}
			pool.Release()
			wg.Done()
		}()
	}
	wg.Wait()

	fmt.Println("TOOK BPOOL", ts.Get())

	fmt.Println("SSS")
}*/

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

var ttT_total = uint32(0)

func (this *Allocator) Allocate(size int) []byte {

	if size <= 96 {
		return make([]byte, 0, size)
	}

	atomic.AddUint32(&ttT_total, 1)

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

func init() {

	oom_sum := 0
	for k, v := range mem_chunks {
		fmt.Println(k, "Full:", v.stat_alloc_full, "Full Small:", v.stat_alloc_full_small,
			"Tail:", v.stat_alloc_tail, "OOM:", v.stat_oom, "Routed:", v.stat_routed,
			"Slab taken:", v.used_slab_count)
		oom_sum += v.stat_oom
	}

	return

	/*fmt.Println("SSSSSSSSSS")

	runtime.GOMAXPROCS(16)
	ok := true

	var mm runtime.MemStats
	runtime.ReadMemStats(&mm)
	fmt.Println(mm)
	fmt.Print(mm)

	ts := NewTimeSpan()

	f, _ := os.Create("cpuprofile")
	pprof.StartCPUProfile(f)

	var wg sync.WaitGroup
	for ii := 0; ii < 20; ii++ {
		wg.Add(1)

		go func(ii int) {

			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			a := MakeAllocator()
			dtot := make([][]int, 0)

			for i := 0; i < 40000 && ok; i++ {

				size := r.Int()%20000 + 100
				_data := a.Allocate(size)
				for e := 0; e < size; e++ {
					_data[e] = ii
				}
				dtot = append(dtot, _data)

				if i%((ii*ii+3)/2) == 0 {
					for _, iv := range dtot {
						for _, iiv := range iv {
							if iiv != ii {
								fmt.Println("!=", iiv, ii)
								ok = false
							}
						}
					}
					dtot = make([][]int, 0)
					a.Release()
				}
			}
			a.Release()
			wg.Done()
		}(ii)
	}
	wg.Wait()

	pprof.StopCPUProfile()



	if !ok {
		fmt.Println("ERROR-X")
	}

	fmt.Println("-----------------------------", "SUM OOM:", oom_sum, " TIME:", ts.Get())*/
}
