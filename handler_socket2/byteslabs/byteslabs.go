package byteslabs

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const mem_chunks_count = 6
const slab_size = 40000
const slab_count = 80

const intpool_size = 100

var intpool *ipool

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

var mutex_slab sync.Mutex

var mem_chunks [mem_chunks_count]*mem_chunk

func init() {

	intpool = MakePool(intpool_size, slab_count)
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
func (this *Allocator) take_additional() {

	// we won't create additional allocator if the allocator is additional already
	if this.is_additional || this.addl_allocator != nil {
		return
	}

	best_slab := -1
	chunks_used := -1
	for i := 0; i < mem_chunks_count; i++ {

		if mem_chunks[i] == this.mem_chunk {
			continue
		}

		_cprobe := int(mem_chunks[i].used_slab_count)

		if chunks_used == -1 || _cprobe < chunks_used {
			best_slab = i
			chunks_used = _cprobe
		}
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

	for _, v := range this.slab_used {
		mem_chunk.slab_used[v] = false
	}

	//fmt.Println("RELEASED", this.slab_used)
	atomic.AddInt32(&mem_chunk.used_slab_count, int32(-len(this.slab_used)))
	this.slab_free_space = []int{}
	this.slab_used = []int{}

	aa_release := this.addl_allocator
	this.addl_allocator = nil
	mem_chunk.mu.Unlock()

	if aa_release != nil {
		aa_release.Release()
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

	slab_free := (slab_size - (size % slab_size)) % slab_size
	slabs_needed := size / slab_size
	if slab_free > 0 {
		slabs_needed++
	}

	mem_chunk := this.mem_chunk
	mem_chunk.mu.Lock()
	defer mem_chunk.mu.Unlock()

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

	//	return mem_chunk.memory[start_pos : start_pos+size]
	if this.is_additional == false {
		this.take_additional()
	}

	if this.addl_allocator != nil {
		mem_chunk.mu.Unlock()
		ret := this.addl_allocator.Allocate(size)

		mem_chunk.mu.Lock()
		mem_chunk.stat_routed++
		return ret
	}

	mem_chunk.stat_oom++
	return make([]byte, 0, size)
}

type TimeSpan struct {
	req_time int64
}

func NewTimeSpan() *TimeSpan {

	ts := TimeSpan{}
	ts.req_time = time.Now().UnixNano()
	return &ts

}

func (ts *TimeSpan) Get() string {

	t := float64((time.Now().UnixNano() - ts.req_time)) / float64(1000000)

	return fmt.Sprintf("%.3fms", t)
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

type ipool struct {
	elements     [][]int
	mu           sync.Mutex
	initial_size int
	element_size int

	reqs, overflows int
}

func MakePool(initial_size, element_size int) *ipool {

	elements := make([][]int, initial_size)
	for i := 0; i < initial_size; i++ {
		elements[i] = make([]int, 0, element_size)
	}

	return &ipool{elements: elements, initial_size: initial_size, element_size: element_size}
}

func (this *ipool) push(val []int) {

	if cap(val) != this.element_size {
		return
	}
	val = val[:0]

	this.mu.Lock()
	defer this.mu.Unlock()
	if len(this.elements) >= this.initial_size {
		return
	}

	this.elements = append(this.elements, val)
}

func (this *ipool) pop() []int {

	this.mu.Lock()
	defer this.mu.Unlock()

	l := len(this.elements)

	if l <= 0 {
		this.overflows++
		return []int{}
	}

	ret := this.elements[l-1]
	this.elements = this.elements[0 : l-1]
	this.reqs++

	return ret
}

func (this *ipool) status() string {
	this.mu.Lock()
	defer this.mu.Unlock()
	return fmt.Sprintf("INTPOOL: Size %d / Used %d ... Requests %d / Overflows %d",
		this.initial_size, this.initial_size-len(this.elements),
		this.reqs, this.overflows)
}
