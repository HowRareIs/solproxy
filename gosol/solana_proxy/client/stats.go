package client

import (
	"fmt"
)

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

func (this *SOLClient) _statsIsDead() (bool, int, string) {

	stat_requests := 0
	stat_errors := 0
	_pos := this.stat_last_60_pos
	for i := 0; i < probe_isalive_seconds; i++ {
		stat_requests += this.stat_last_60[_pos].stat_done
		stat_errors += this.stat_last_60[_pos].stat_error_resp

		_pos-- // take current second into account
		if _pos < 0 {
			_pos = 59
		}
	}

	// make sure we have some requests, if not declare we're dead only in non-conserve-requests mode
	if stat_requests < probe_ok_min_requests && this.attr&CLIENT_CONSERVE_REQUESTS == 0 {
		log := fmt.Sprintf("Last %ds, have %d request(s) %d required",
			probe_isalive_seconds, stat_requests, probe_ok_min_requests)
		return true, stat_requests, log
	}

	// if we have no requests we assume something is wrong and we mark the node as dead
	// if there are no requests but we're in conserving mode, let's use what we have
	dead := stat_errors*5 > stat_requests
	log := fmt.Sprintf("Last %ds, Requests: %d, Errors: %d", probe_isalive_seconds, stat_requests, stat_errors)
	return dead, stat_requests, log
}
