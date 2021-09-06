package compress

import (
	"bytes"
	"compress/flate"
	"fmt"
	"sync"

	"github.com/slawomir-pryczek/handler_socket2/byteslabs"
)

const max_compressors = 20

var mu sync.Mutex
var f_writers []*flate.Writer
var fw_stat_compressors_reuse = 0
var fw_stat_compressors_created = 0
var fw_stat_underflows = 0
var fw_stat_overflows = 0
var fw_stat_underflows_b = 0
var fw_stat_overflows_b = 0

func init() {
	f_writers = make([]*flate.Writer, max_compressors)
	for i := 0; i < max_compressors; i++ {
		tmp, _ := flate.NewWriter(nil, 2)
		f_writers[i] = tmp
	}
}

func CompressSimple(data []byte, alloc *byteslabs.Allocator) []byte {

	mu.Lock()
	var writer *flate.Writer = nil
	if len(f_writers) > 0 {
		writer = f_writers[len(f_writers)-1]
		f_writers = f_writers[:len(f_writers)-1]
		fw_stat_compressors_reuse++
	} else {
		fw_stat_compressors_created++
	}
	mu.Unlock()
	if writer == nil {
		writer, _ = flate.NewWriter(nil, 2)
	}

	_d_len := len(data)
	_d_len = int(float32(_d_len) / 1.3)

	b := bytes.NewBuffer(alloc.Allocate(_d_len))
	writer.Reset(b)
	writer.Write(data)
	writer.Close()

	diff := _d_len - b.Len()
	if diff >= 0 {
		fw_stat_underflows++
		fw_stat_underflows_b += diff
	} else {
		fw_stat_overflows++
		fw_stat_overflows_b += -diff
	}

	mu.Lock()
	f_writers = append(f_writers, writer)
	mu.Unlock()

	return b.Bytes()
}

func CompressSimpleStatus() string {

	ret := ""

	mu.Lock()
	defer mu.Unlock()

	ret += fmt.Sprintf("FastCompress*: %d, SlowCompress**: %d    PreAllocated Compressors: %d\n",
		fw_stat_compressors_reuse, fw_stat_compressors_created, max_compressors)
	ret += "  * Compressors that are re-used without memory re-allocation, ** Compressors fully re-allocated\n"

	_u := float64(fw_stat_underflows_b) / float64(fw_stat_underflows)
	_o := float64(fw_stat_overflows_b) / float64(fw_stat_overflows)
	if fw_stat_underflows == 0 {
		_u = 0
	}
	if fw_stat_overflows == 0 {
		_o = 0
	}
	ret += fmt.Sprintf("Buffer Predictions - Underflows: %d / %.1fKB Underflow Per Request\n", fw_stat_underflows, _u/float64(1024.0))
	ret += fmt.Sprintf("Buffer Predictions - Overflows: %d / %.1fKB Overflow Per Request\n", fw_stat_overflows, _o/float64(1024.0))
	ret += "     Underflows mean that we allocated too large buffer from SLAB, this is normal situation.\n"
	ret += "     Overflow means that we needed to re-allocate compression buffer because there was too little space\n"

	return ret
}
