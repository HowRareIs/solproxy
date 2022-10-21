package hscommon

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// sorting
type ScoredItems struct {
	Item  string
	Score int64
}

type SIArr []ScoredItems

func (si SIArr) Len() int {
	return len(si)
}
func (si SIArr) Less(i, j int) bool {
	return si[i].Score < si[j].Score
}
func (si SIArr) Swap(i, j int) {
	si[i], si[j] = si[j], si[i]
}

type StrScoredItems struct {
	item     string
	score    string
	_usrdata []string
}

type SSIArr []StrScoredItems

func (si SSIArr) Len() int {
	return len(si)
}
func (si SSIArr) Less(i, j int) bool {
	return si[i].score < si[j].score
}
func (si SSIArr) Swap(i, j int) {
	si[i], si[j] = si[j], si[i]
}

func FormatBytes(b uint64) string {

	if b < 0 {
		return "-"
	}

	units := []string{"B", "KB", "MB", "GB", "TB"}

	_unit_num := int(math.Floor(math.Min(math.Log(float64(b))/math.Log(1024), float64(len(units)-1))))
	if _unit_num <= 0 || _unit_num >= len(units) {
		return fmt.Sprintf("%dB", b)
	}

	unit := units[_unit_num]
	_p := uint64(math.Pow(1024, float64(_unit_num)))

	var ba, bb uint64
	if _unit_num > 2 {
		ba = uint64(b * 1000 / uint64(_p))
		bb = ba % 1000
		ba = ba / 1000
		return fmt.Sprintf("%d.%03d%s", ba, bb, unit)
	} else {
		ba = uint64(b * 100 / uint64(_p))
		bb = ba % 100
		ba = ba / 100
	}

	return fmt.Sprintf("%d.%02d%s", ba, bb, unit)
}

func FormatBytesI(b uint64) string {
	ret := strings.Split(FormatBytes(b), ".")
	if len(ret) < 2 || len(ret[1]) < 2 {
		return strings.Join(ret, ".")
	}

	return ret[0] + "." + string(ret[1][0]) + strings.Trim(ret[1], "0123456789)")
}

func FormatTime(t int) string {
	uptime_str := ""
	ranges := []string{"day", "hour", "minute", "second"}
	div := []int{60 * 60 * 24, 60 * 60, 60, 1}

	for i := 0; i < 4; i++ {

		u_ := t / div[i]
		s_ := ""
		if u_ > 1 {
			s_ = "s"
		}

		if u_ > 0 {
			uptime_str += fmt.Sprintf("%d %s%s", u_, ranges[i], s_)
			t = t % div[i]
		}
	}

	return uptime_str
}

type tablegen struct {
	header    string
	items     []StrScoredItems
	className string

	_header_data []string
	_row_data    [][]string
}

func NewTableGen(header ...string) *tablegen {
	ret := tablegen{}
	ret.header = "<tr><td>" + strings.Join(header, "</td><td>") + "</td></tr>"
	ret._header_data = header
	return &ret
}

func (this *tablegen) AddRow(data ...string) {
	_d := "<tr class='##class##'><td>" + strings.Join(data, "</td><td>") + "</td></tr>\n"
	this.items = append(this.items, StrScoredItems{item: _d, _usrdata: data})

	this._row_data = append(this._row_data, data)
}

func (this *tablegen) Render() string {

	data := ""
	for pos, v := range this.items {
		_class := "r" + strconv.Itoa(pos%2)
		data += strings.Replace(v.item, "##class##", _class, 1)
	}

	_class := ""
	if len(this.className) > 0 {
		_class = " class='" + this.className + "'"
	}

	return "<table" + _class + ">\n<thead>" + this.header + "</thead>\n" + "<tbody>" + data + "</tbody>\n</table>"
}

func (this *tablegen) RenderHorizFlat(columns int) string {

	ret := ""
	_class := ""
	if len(this.className) > 0 {
		_class = " class='" + this.className + "'"
	}

	// do we have class
	has_class := this._header_data[len(this._header_data)-1] == "_class"
	if has_class {
		this._header_data = this._header_data[0 : len(this._header_data)-1]
	}

	pos := 0
	ret += "<table" + _class + ">"
	for pos < len(this._row_data) {
		for col_no, v := range this._header_data {

			ret += "<tr><td>" + v + "</td>"
			for i := pos; i < len(this._row_data) && i-pos < columns; i++ {

				class := ""
				if has_class && len(this._row_data[i]) >= len(this._header_data)+1 {
					class = " class='" + this._row_data[i][len(this._header_data)] + "'"
				}

				if col_no >= len(this._row_data[i]) {
					break
				}

				ret += "<td" + class + ">" + this._row_data[i][col_no] + "</td>"
			}
			ret += "</tr>"
		}
		pos += columns
	}
	ret += "</table>"

	return ret
}

func (this *tablegen) RenderHoriz(columns int) string {

	ret := ""
	_class := ""
	if len(this.className) > 0 {
		_class = " class='" + this.className + "'"
	}

	pos := 0

	for pos < len(this._row_data) {

		ret += "<table" + _class + "><thead>"
		for col_no, v := range this._header_data {
			ret += "<tr><td>" + v + "</td>"
			for i := pos; i < len(this._row_data) && i-pos < columns; i++ {

				if col_no >= len(this._row_data[i]) {
					break
				}

				ret += "<td>" + this._row_data[i][col_no] + "</td>"
			}

			if col_no == 0 {
				ret += "</thead>"
			}
			ret += "</tr>"
		}
		ret += "</table><br><br>"

		pos += columns
	}

	return ret

}

func (this *tablegen) RenderSorted(columns ...int) string {

	for pos, v := range this.items {

		_score := ""
		for _, kcol := range columns {
			if len(v._usrdata) > kcol {
				_score += (v._usrdata[kcol] + "-")
			}
		}

		this.items[pos].score = _score
	}

	// sort the data
	sort.Sort(SSIArr(this.items))

	// we have re-sorted data, now render normally
	return this.Render()
}

func (this *tablegen) RenderSortedByInt(columns ...int) string {

	sort.Slice(this.items, func(i, j int) bool {

		for _, col := range columns {
			a, _ := strconv.Atoi(this.items[i]._usrdata[col])
			b, _ := strconv.Atoi(this.items[j]._usrdata[col])

			if a != b {
				return a < b
			}
		}

		return false
	})

	// we have re-sorted data, now render normally
	return this.Render()
}

func (this *tablegen) RenderSortedRaw(scores []string) string {

	slen := len(scores)
	for pos, _ := range this.items {

		if pos < slen {
			this.items[pos].score = scores[pos]
		}
	}

	// sort the data
	sort.Sort(SSIArr(this.items))

	// we have re-sorted data, now render normally
	return this.Render()
}

func (this *tablegen) SetClass(className string) {
	this.className = className
}

type Buffer struct {
	b []byte
}

func NewBuffer(b []byte) *Buffer {
	return &Buffer{b[:0]}
}
func (this *Buffer) WriteStr(a string) {

	aa := []byte(a)
	start := len(this.b)
	data_len := len(aa)

	copy(this.b[start:start+data_len], aa)
	this.b = this.b[0 : start+data_len]

}

func (this *Buffer) Bytes() []byte {
	return this.b
}

// #############################################################################
// Current timestamp functionality
var unixTS int64 = 0

func init() {

	go func() {

		var __tmp int64 = 0
		var __tmp2 int64 = 0
		for {

			__tmp = time.Now().Unix()
			if __tmp != __tmp2 {
				__tmp2 = __tmp
				atomic.StoreInt64(&unixTS, __tmp)
			}

			time.Sleep(100 * time.Millisecond)
		}

	}()
}

// get current timestamp in seconds
func TSNow() int {

	__tmp := atomic.LoadInt64(&unixTS)

	// maybe the thread isn't running yet
	if __tmp == 0 {
		__tmp = time.Now().Unix()
	}

	return int(__tmp)
}

// generate perc used
type bucket_stats struct {
	size []int
	used []int

	elements int
	curr_el  int
}

func NewBucketStats(elements int) *bucket_stats {

	return &bucket_stats{make([]int, elements), make([]int, elements), elements, 0}
}

func (this *bucket_stats) Push(size, used int) {

	this.size[this.curr_el] += size
	this.used[this.curr_el] += used

	this.curr_el++
	if this.curr_el >= this.elements {
		this.curr_el = 0
	}
}

func (this *bucket_stats) Gen() string {

	ret := ""
	for i := 0; i < this.elements; i++ {

		size := this.size[i]
		used := this.used[i]
		if size == 0 || used == 0 {
			ret += "_"
			continue
		}

		perc := int(float64(used*100.0) / float64(size))

		switch {
		case perc < 2:
			ret += "&#x25E1;"
		case perc < 4:
			ret += "&#x25CC;"
		case perc < 8:
			ret += "&#x25CB;"
		case perc < 12:
			ret += "&#x25CE;"
		case perc < 25:
			ret += "&#x25D4;"
		case perc < 50:
			ret += "&#x25D1;"
		case perc < 75:
			ret += "&#x25D5;"
		case perc < 90:
			ret += "&#x1f311;"
		case perc >= 90:
			ret += "<span style='color:red'>&#x1f311;</span>"
		}

	}

	return "<pre style='font-family: monospace!important'> (?) " + ret + "</pre>"
}

// generate perc used
type percentile_stats struct {
	data   []int
	sum    int
	sorted bool
}

func NewPercentileStats(maxSize int) *percentile_stats {
	return &percentile_stats{make([]int, 0, maxSize), 0, true}
}

func (this *percentile_stats) Push(v int) {
	this.data = append(this.data, v)
	this.sum += v
	this.sorted = false
}

func (this *percentile_stats) Get(percentile int) int {

	if len(this.data) < 5 {
		return 0
	}

	if !this.sorted {
		sort.Sort(sort.IntSlice(this.data))
		this.sorted = true
	}

	l := int(float64(len(this.data)) * (float64(percentile) / float64(100.0)))
	if l >= len(this.data) {
		l = len(this.data) - 1
	}

	return this.data[l]
}

func (this *percentile_stats) Avg() float64 {
	return float64(this.sum) / float64(len(this.data))
}

func (this *percentile_stats) Clean() {
	this.data = this.data[:0]
	this.sum = 0
	this.sorted = true
}

func (this *percentile_stats) CountLoHi(threshold int) (int, int) {

	if len(this.data) < 1 {
		return 0, 0
	}

	if !this.sorted {
		sort.Sort(sort.IntSlice(this.data))
		this.sorted = true
	}

	for k, v := range this.data {

		if v > threshold {
			return k, len(this.data) - k
		}

	}

	return len(this.data), 0
}

func Inet_aton(ip string) uint32 {

	var ret uint32

	chunks := strings.Split(strings.Trim(ip, "\r\n \t"), ".")
	for k, v := range chunks {

		if k > 3 {
			break
		}

		vv, _ := strconv.Atoi(v)
		k = (3 - k) * 8

		ret = ret + (uint32(vv) << uint8(k))
	}

	return ret
}

func Inet_ntoa(ip uint32) string {

	ip_oct := [4]byte{0, 0, 0, 0}
	for i := 0; i < 4; i++ {
		ip_oct[3-i] = byte(ip & 0xff)
		ip = ip >> 8
	}

	return fmt.Sprintf("%d.%d.%d.%d", ip_oct[0], ip_oct[1], ip_oct[2], ip_oct[3])
}

// #############################################################################
// Time Span

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

func (ts *TimeSpan) GetUS() string {

	t := float64((time.Now().UnixNano() - ts.req_time)) / float64(1000)

	return fmt.Sprintf("%.3fus", t)
}

func (ts *TimeSpan) GetRaw() float64 {
	return float64((time.Now().UnixNano() - ts.req_time)) / float64(1000000)
}

func (ts *TimeSpan) GetRawUS() float64 {
	return float64((time.Now().UnixNano() - ts.req_time)) / float64(1000)
}

func ExitErr(err string) {

	fmt.Fprintf(os.Stderr, "Fatal error, server will shutdown\n"+err+"\n")
	os.Exit(1)
}
