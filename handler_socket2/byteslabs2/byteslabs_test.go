package byteslabs2

import (
	"bytes"
	"fmt"
	"math/rand"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var rnd_gen []int
var rnd_pos int32

func init() {
	rnd_gen = make([]int, 50000)
	for i := 0; i < len(rnd_gen); i++ {
		rnd_gen[i] = rand.Int()
	}
}

func rand_b(n int) int {
	pos := int(atomic.AddInt32(&rnd_pos, 1))
	return rnd_gen[pos%len(rnd_gen)] % n
}

func TestByteslabs(t *testing.T) {

	ts := time.Now().UnixMilli()

	var letterRunes = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	var randomStr = []byte(nil)
	rnd_init := func(n int) []byte {
		b := make([]byte, n)
		for i := range b {
			b[i] = letterRunes[rand_b(len(letterRunes))]
		}
		return b
	}
	randomStr = rnd_init(100000)
	rnd := func(n int) []byte {
		start := rand.Intn(1000)
		return randomStr[start : start+n]
	}

	runtime.GOMAXPROCS(1)

	bsm := Make(8, 40000, 100)

	single_pass := func(disp bool) {
		alloc := bsm.MakeAllocator()

		els_ok, els_fail, els_len := 0, 0, 0
		stored := make([][]byte, 0, 0)
		n_strings := rand.Intn(200) + 10

		for i := 0; i < n_strings; i++ {
			n := 0
			if i%10 == 0 {
				n = rand.Intn(60000) + 30000
			} else {
				n = rand.Intn(600)
			}

			tostore := rnd(n)
			els_len += n
			stored = append(stored, tostore)
			if i%2 == 0 {
				// store test1
				//_sa := make([]byte, n)
				_sa := alloc.Allocate(n)[0:n]
				copy(_sa, tostore)
				stored = append(stored, _sa)
			} else {
				//_sa := make([]byte, 0, n)
				_sa := alloc.Allocate(n)
				_sa_b := _sa[0:n]
				for _, b := range tostore {
					_sa = append(_sa, b)
				}
				stored = append(stored, _sa_b)
			}

			for i := 0; i < len(stored); i += 2 {
				ok := bytes.Compare(stored[i], stored[i+1]) == 0
				if ok {
					els_ok++
				} else {
					els_fail++
					fmt.Println("!! ", len(stored[i]), len(stored[i+1]))
				}
			}
		}
		alloc.Release()
		if disp {
			fmt.Println(n_strings, "Strings... ", "OK: ", els_ok, "  Fail", els_fail, "   Bytes: ", els_len)
		}

	}

	threads := 10
	for z := 0; z < 60/threads; z++ {
		wg := sync.WaitGroup{}
		for i := 0; i < threads; i++ {
			i := i
			wg.Add(1)
			go func() {
				fmt.Println("Started : ", i)
				for i := 0; i < 300; i++ {
					single_pass(i%60 == 0)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}

	fmt.Println(float64(time.Now().UnixMilli()-ts) / 1000)

	var garC debug.GCStats
	debug.ReadGCStats(&garC)
	fmt.Printf("\nLastGC:\t%s", garC.LastGC)         // time of last collection
	fmt.Printf("\nNumGC:\t%d", garC.NumGC)           // number of garbage collections
	fmt.Printf("\nPauseTotal:\t%s", garC.PauseTotal) // total pause for all collections

}

func BenchmarkByteslabs(b *testing.B) {

	ts := time.Now().UnixMilli()
	runtime.GOMAXPROCS(60)

	bsm := Make(8, 40000, 100)
	single_pass := func(disp bool) {
		alloc := bsm.MakeAllocator()
		n_strings := rand_b(200) + 10

		for i := 0; i < n_strings; i++ {
			n := 0
			if i%10 == 0 {
				n = rand_b(60000) + 30000
			} else {
				n = rand_b(600)
			}
			alloc.Allocate(n)
		}
		alloc.Release()
	}

	threads := 60
	fmt.Println("Startig threads: ")
	for z := 0; z < 60/threads; z++ {
		wg := sync.WaitGroup{}
		for i := 0; i < threads; i++ {
			i := i
			wg.Add(1)
			go func() {
				fmt.Print(i, " ")
				for i := 0; i < 1000; i++ {
					single_pass(i%60 == 0)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
	fmt.Println()

	fmt.Println(float64(time.Now().UnixMilli()-ts) / 1000)

	var garC debug.GCStats
	debug.ReadGCStats(&garC)
	fmt.Printf("\nLastGC:\t%s", garC.LastGC)         // time of last collection
	fmt.Printf("\nNumGC:\t%d", garC.NumGC)           // number of garbage collections
	fmt.Printf("\nPauseTotal:\t%s", garC.PauseTotal) // total pause for all collections

	b.ReportAllocs()
	fmt.Println(bsm.GetStatusStr())
}
