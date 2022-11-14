package stats

import (
	"github.com/slawomir-pryczek/HSServer/handler_socket2/hscommon"
)

type uh_struct struct {
	timestamp int

	connections   uint64
	requests      uint64
	errors        uint64
	req_time      uint64
	req_time_full uint64

	b_request_size       uint64
	b_request_compressed uint64

	b_resp_size       uint64
	b_resp_compressed uint64

	resp_skipped   uint64
	resp_b_skipped uint64
}

var uptime_history [6]uh_struct

func uh_add_unsafe(now int, connections, requests, errors, req_time, req_time_full, b_request_size, b_request_compressed, b_resp_size, b_resp_compressed uint64, skip_response_sent bool) {

	uh_now := &uptime_history[now%6]
	if uh_now.timestamp != now {
		uh_now.timestamp = now
		uh_now.connections = connections
		uh_now.requests = requests
		uh_now.errors = errors
		uh_now.req_time = req_time
		uh_now.req_time_full = req_time_full

		uh_now.b_request_size = b_request_size
		uh_now.b_request_compressed = b_request_compressed
		uh_now.b_resp_size = b_resp_size
		uh_now.b_resp_compressed = b_resp_compressed

		if skip_response_sent {
			uh_now.resp_skipped = requests
			uh_now.resp_b_skipped = b_resp_compressed
		} else {
			uh_now.resp_skipped, uh_now.resp_b_skipped = 0, 0
		}

	} else {
		uh_now.connections += connections
		uh_now.requests += requests
		uh_now.errors += errors
		uh_now.req_time += req_time
		uh_now.req_time_full += req_time_full

		uh_now.b_request_size += b_request_size
		uh_now.b_request_compressed += b_request_compressed
		uh_now.b_resp_size += b_resp_size
		uh_now.b_resp_compressed += b_resp_compressed

		if skip_response_sent {
			uh_now.resp_skipped += requests
			uh_now.resp_b_skipped += b_resp_compressed
		}
	}

}

func uh_get() uh_struct {

	ret := uh_struct{}
	now := hscommon.TSNow()

	stats_mutex.Lock()
	for i := 0; i < 6; i++ {

		// current second - we're still collecting stats
		if i == 0 {
			continue
		}

		now--
		uh_now := &uptime_history[now%6]
		if uh_now.timestamp != now {
			uh_now.timestamp = now
			uh_now.connections = 0
			uh_now.requests = 0
			uh_now.errors = 0
			uh_now.req_time = 0
			uh_now.req_time_full = 0

			uh_now.b_request_size = 0
			uh_now.b_request_compressed = 0
			uh_now.b_resp_size = 0
			uh_now.b_resp_compressed = 0

			uh_now.resp_skipped = 0
			uh_now.resp_b_skipped = 0
			continue
		}

		ret.connections += uh_now.connections
		ret.requests += uh_now.requests
		ret.errors += uh_now.errors
		ret.req_time += uh_now.req_time
		ret.req_time_full += uh_now.req_time_full

		ret.b_request_size += uh_now.b_request_size
		ret.b_request_compressed += uh_now.b_request_compressed
		ret.b_resp_size += uh_now.b_resp_size - uh_now.resp_b_skipped
		ret.b_resp_compressed += uh_now.b_resp_compressed - uh_now.resp_b_skipped

		ret.resp_skipped += uh_now.resp_skipped
		ret.resp_b_skipped += uh_now.resp_b_skipped
	}
	stats_mutex.Unlock()
	return ret
}
