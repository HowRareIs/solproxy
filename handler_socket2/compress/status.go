package compress

import (
	"fmt"
	"sync"
	"time"

	"github.com/slawomir-pryczek/HSServer/handler_socket2/hscommon"
)

const thread_stopped_symbol = 'x'

type stat_items struct {
	compression_count  int
	compression_pieces int
	data_in_size       uint64
	data_out_size      uint64

	buffer_too_small          int
	compression_ratio_too_low int
}

type stat struct {
	name       string
	s_totals   stat_items
	s_last_70  [7]stat_items
	curr_slice int

	mu sync.Mutex

	thread_symbol     []byte
	thread_running    []byte
	thread_items_done []int
}

func MakeStat(name string) *stat {

	ret := &stat{}
	ret.name = name
	ret.thread_symbol = make([]byte, 0, 30)
	ret.thread_running = make([]byte, 0, 30)
	go func() {

		last_slice := -1
		for {
			time.Sleep(500 * time.Millisecond)
			slice := int(time.Now().Unix()/10) % 7
			if slice == last_slice {
				continue
			}
			last_slice = slice

			ret.mu.Lock()
			ret.curr_slice = slice
			ret.s_last_70[slice] = stat_items{}
			ret.mu.Unlock()
		}
	}()

	return ret
}

func (this *stat) ThreadAdd(symbol byte) int {
	this.mu.Lock()
	this.thread_symbol = append(this.thread_symbol, symbol)
	this.thread_running = append(this.thread_running, thread_stopped_symbol)
	this.thread_items_done = append(this.thread_items_done, 0)
	ret := len(this.thread_symbol) - 1
	this.mu.Unlock()
	return ret
}

func (this *stat) ThreadSetRunning(thread_id int, is_running bool) {
	this.mu.Lock()
	if is_running {
		this.thread_running[thread_id] = this.thread_symbol[thread_id]
	} else {
		this.thread_running[thread_id] = thread_stopped_symbol
	}
	this.mu.Unlock()
}

func (this *stat) doStats(thread_id, piece_no, data_in_size, data_out_size int) {
	this.mu.Lock()
	if piece_no == 0 {
		this.s_totals.compression_count++
	}
	this.s_totals.compression_pieces++
	this.s_totals.data_in_size += uint64(data_in_size)
	this.s_totals.data_out_size += uint64(data_out_size)

	_slice := this.curr_slice
	if piece_no == 0 {
		this.s_last_70[_slice].compression_count++
	}
	this.s_last_70[_slice].compression_pieces++
	this.s_last_70[_slice].data_in_size += uint64(data_in_size)
	this.s_last_70[_slice].data_out_size += uint64(data_out_size)
	this.thread_running[thread_id] = thread_stopped_symbol
	this.thread_items_done[thread_id]++
	this.mu.Unlock()
}

func (this *stat) reportBufferTooSmall() {
	this.mu.Lock()
	this.s_totals.buffer_too_small++
	this.s_last_70[this.curr_slice].buffer_too_small++
	this.mu.Unlock()
}

func (this *stat) reportCompressionRatioTooLow() {
	this.mu.Lock()
	this.s_totals.compression_ratio_too_low++
	this.s_last_70[this.curr_slice].compression_ratio_too_low++
	this.mu.Unlock()
}

func (this *stat) GetStatus() string {

	table := hscommon.NewTableGen("Time", "Compressions", "Pieces", "Data In", "Data Out", "Ratio", "E-RLow", "E-Buffer")
	table.SetClass("tab threads")

	this.mu.Lock()

	_num := func(data ...int) string {
		if len(data) == 1 {
			return fmt.Sprintf("%d", data[0])
		}
		zeros := fmt.Sprintf("%d", data[1])
		return fmt.Sprintf("%0"+zeros+"d", data[0])
	}
	_addrow := func(title string, i stat_items) {
		ratio := " - "
		if i.data_out_size > 0 {
			__r := (i.data_in_size * 100) / i.data_out_size
			ratio = fmt.Sprintf("x%.2f", float64(__r)/100)
		}
		table.AddRow(title, _num(i.compression_count), _num(i.compression_pieces),
			hscommon.FormatBytes(i.data_in_size), hscommon.FormatBytes(i.data_out_size), ratio,
			_num(i.compression_ratio_too_low), _num(i.buffer_too_small))

	}

	i_last_10 := stat_items{}
	i_last_60 := stat_items{}
	for i := 1; i < len(this.s_last_70); i++ {
		pos := (this.curr_slice + len(this.s_last_70) - i) % len(this.s_last_70)

		sr := this.s_last_70[pos]
		if i == 1 {
			i_last_10 = sr
		}
		i_last_60.buffer_too_small += sr.buffer_too_small
		i_last_60.compression_count += sr.compression_count
		i_last_60.compression_pieces += sr.compression_pieces
		i_last_60.compression_ratio_too_low += sr.compression_ratio_too_low
		i_last_60.data_in_size += sr.data_in_size
		i_last_60.data_out_size += sr.data_out_size
	}
	table.AddRow(this.name)
	_addrow("Last 10s", i_last_10)
	_addrow("Last 60s", i_last_60)
	_addrow("Total", this.s_totals)

	tab_threads := hscommon.NewTableGen("Thread", "Items Done", "_class")
	tab_threads.SetClass("tab compressing")

	_add_thread := func(thr_num int, thr_status byte, thr_type_filter byte) {
		if thr_type_filter != this.thread_symbol[thr_num] {
			return
		}
		class := ""
		status := "&#x26ab; "
		if thr_status != 'x' {
			status = "&#x25b6; "
			class = "running"
		}
		status += _num(thr_num, 3)
		status += " (" + string(this.thread_symbol[thr_num]) + ")"
		tab_threads.AddRow(status, _num(this.thread_items_done[thr_num]), class)
	}

	for thr_num, status := range this.thread_running {
		_add_thread(thr_num, status, 'N')
	}
	for thr_num, status := range this.thread_running {
		_add_thread(thr_num, status, 'F')
	}
	this.mu.Unlock()

	ret := table.Render()
	ret += tab_threads.RenderHorizFlat(14)
	return ret
}
