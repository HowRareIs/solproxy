package stats

import (
	"fmt"
	"sync"
	"time"

	"github.com/slawomir-pryczek/handler_socket2/hscommon"
)

// #############################################################################
// general stats

var global_stats_connections uint64 = 0
var global_stats_requests uint64 = 0
var global_stats_errors uint64 = 0
var global_stats_req_time uint64 = 0
var global_stats_req_time_full uint64 = 0 // time of whole request (including received + sent)

// #############################################################################
// statistical structutes

type Connection struct {
	id         uint64
	ip         string
	status     string
	action     string
	data       string
	comment    string
	is_error   bool
	start_time int64
	end_time   int64

	request_size, request_size_compressed, response_size uint64
}

type stats struct {
	requests      uint64
	req_time      uint64
	req_time_full uint64

	b_request_size       uint64
	b_request_compressed uint64

	b_resp_size       uint64
	b_resp_compressed uint64

	resp_skipped   uint64
	resp_b_skipped uint64
}

func (this *stats) getAdjStats() stats {

	ret := stats{}
	ret.requests = this.requests
	ret.req_time = this.req_time
	ret.req_time_full = this.req_time_full

	ret.b_request_size = this.b_request_size
	ret.b_request_compressed = this.b_request_compressed

	ret.b_resp_size = this.b_resp_size - this.resp_b_skipped
	ret.b_resp_compressed = this.b_resp_compressed - this.resp_b_skipped

	ret.resp_skipped = this.resp_skipped
	ret.resp_b_skipped = this.resp_b_skipped

	return ret
}

var stats_mutex sync.Mutex
var status_connections map[uint64]*Connection = make(map[uint64]*Connection)
var stats_actions map[string]*stats = make(map[string]*stats)

// #############################################################################
// statistical structutes

type Conn struct {
	conn_id uint64
}

const TYPE_NUMERIC_STAT = 0
const TYPE_CONN_OPEN = 0
const TYPE_CONN_CLOSE = 0
const TYPE_CONN_UPDATE = 0

type stat_event struct {
	e_type  int
	conn_id uint64

	str1 string
	str2 string
	data [9]int
}

func MakeConnection(remoteAddr string) *Connection {

	now_us := time.Now().UnixNano()

	newconn := &Connection{}
	newconn.ip = remoteAddr
	newconn.status = "O"
	newconn.start_time = now_us
	newconn.end_time = 0

	stats_mutex.Lock()
	global_stats_connections++
	conn_id := global_stats_connections
	newconn.id = conn_id
	status_connections[conn_id] = newconn
	uh_add_unsafe(int(now_us/1000000000), 1, 0, 0, 0, 0, 0, 0, 0, 0, false)
	stats_mutex.Unlock()

	return newconn
}

func (this *Connection) StateReading() int64 {

	start := time.Now().UnixNano()

	stats_mutex.Lock()
	this.status = "Sr"
	this.start_time = start
	this.end_time = 0
	stats_mutex.Unlock()

	return start

}

func (this *Connection) StateServing(action, _pinfo string) {

	stats_mutex.Lock()
	this.status = "Ss1"
	this.action = action
	this.data = _pinfo
	stats_mutex.Unlock()
}

func (this *Connection) StateWriting(request_size, request_size_compressed, response_size uint64) uint64 {

	time_end := time.Now().UnixNano()

	stats_mutex.Lock()
	this.status = "W"
	this.end_time = time_end

	this.request_size = request_size
	this.request_size_compressed = request_size_compressed
	this.response_size = response_size

	took := uint64((time_end - this.start_time) / 1000)
	action := this.action

	if _, ok := stats_actions[action]; !ok {
		stats_actions[action] = &stats{}
	}

	_sa := stats_actions[action]
	_sa.requests++
	_sa.req_time += took
	_sa.b_request_size += request_size
	_sa.b_request_compressed += request_size_compressed
	_sa.b_resp_size += response_size
	stats_mutex.Unlock()

	return took
}

func (this *Connection) StateKeepalive(resp_compressed, took uint64, skip_response_sent bool) {

	now_us := time.Now().UnixNano()

	stats_mutex.Lock()
	_took_full := uint64((now_us - this.start_time) / 1000)
	global_stats_requests++
	global_stats_req_time += took
	global_stats_req_time_full += uint64(_took_full)

	action := this.action
	if _, ok := stats_actions[action]; !ok {
		stats_actions[action] = &stats{}
	}
	_sa := stats_actions[action]

	this.status = "K"
	_sa.b_resp_compressed += resp_compressed
	_sa.req_time_full += _took_full
	if skip_response_sent {
		_sa.resp_b_skipped += this.response_size
		_sa.resp_skipped++
	}

	uh_add_unsafe(int(now_us/1000000000), 0, 1, 0, took, _took_full, this.request_size, this.request_size_compressed, this.response_size, resp_compressed, skip_response_sent)
	stats_mutex.Unlock()

}

func (this *Connection) Close(comment string, is_error bool) {

	if is_error {
		stats_mutex.Lock()
		global_stats_errors++
		uh_add_unsafe(hscommon.TSNow(), 0, 0, 1, 0, 0, 0, 0, 0, 0, false)
		stats_mutex.Unlock()
	}

	x := func(conn_id uint64, comment string, is_error bool) {

		if is_error {
			fmt.Print("Error: " + comment)
		} else {
			fmt.Print("Info: " + comment)
		}

		time.Sleep(5000 * time.Millisecond)
		stats_mutex.Lock()
		var c *Connection = status_connections[conn_id]
		if c.end_time == 0 {
			c.start_time = 0
		}
		c.status = "X"
		c.comment = comment
		c.is_error = is_error
		stats_mutex.Unlock()

		time.Sleep(5000 * time.Millisecond)
		stats_mutex.Lock()
		delete(status_connections, conn_id)
		stats_mutex.Unlock()
	}

	go x(this.id, comment, is_error)
}
