package client

import (
	"fmt"
)

type stat struct {
	stat_error_req          int
	stat_error_resp         int
	stat_error_resp_read    int
	stat_error_json_decode  int
	stat_error_json_marshal int
	stat_done               int
	stat_ns_total           uint64

	stat_request_by_fn  map[string]int
	stat_bytes_received int
	stat_bytes_sent     int
}

func (this *SOLClient) _statsGetAggr(seconds int) stat {

	s := stat{}
	_pos := this.stat_last_60_pos
	for i := 0; i < seconds; i++ {
		_pos--
		if _pos < 0 {
			_pos = 59
		}

		_tmp := this.stat_last_60[_pos%60]
		s.stat_done += _tmp.stat_done
		s.stat_error_json_decode += _tmp.stat_error_json_decode
		s.stat_error_json_marshal += _tmp.stat_error_json_marshal
		s.stat_error_req += _tmp.stat_error_req
		s.stat_error_resp += _tmp.stat_error_resp
		s.stat_error_resp_read += _tmp.stat_error_resp_read
		s.stat_ns_total += _tmp.stat_ns_total

		_tmp2 := make(map[string]int)
		for k, v := range _tmp.stat_request_by_fn {
			_tmp2[k] = _tmp2[k] + v
		}
		s.stat_request_by_fn = _tmp2
		s.stat_bytes_received += _tmp.stat_bytes_received
		s.stat_bytes_sent += _tmp.stat_bytes_sent
	}

	return s
}

func (this *SOLClient) _statsIsDead() (bool, int, int, string) {

	probe_isalive_seconds := this._probe_time * 2
	if probe_isalive_seconds < 30 {
		probe_isalive_seconds = 30
	}
	if probe_isalive_seconds > 60 {
		probe_isalive_seconds = 60
	}

	stat_requests := 0
	stat_errors := 0
	_pos := this.stat_last_60_pos
	for i := 0; i < probe_isalive_seconds; i++ {
		stat_requests += this.stat_last_60[_pos].stat_done
		stat_errors += this.stat_last_60[_pos].stat_error_resp
		stat_errors += this.stat_last_60[_pos].stat_error_resp_read
		stat_errors += this.stat_last_60[_pos].stat_error_json_decode

		_pos-- // take current second into account
		if _pos < 0 {
			_pos = 59
		}
	}

	// if we have no requests we assume something is wrong and we mark the node as dead
	// only if we're probing the node
	dead := this._probe_time == 0 && stat_errors*5 > stat_requests
	dead = dead || this._probe_time > 0 && stat_errors*5 >= stat_requests

	log := fmt.Sprintf("Health probing time %ds every %ds, Requests: %d, Errors: %d", probe_isalive_seconds, this._probe_time,
		stat_requests, stat_errors)
	return dead, stat_requests, stat_errors, log
}
