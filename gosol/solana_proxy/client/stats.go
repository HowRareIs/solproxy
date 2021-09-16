package client

import (
	"gosol/solana_proxy/throttle"
)

func (this *SOLClient) GetThrottle() throttle.Throttle {
	_a, _b, _c := this._statsGetThrottle()
	return throttle.Make(this.is_public_node, _a, _b, _c)
}

func (this *SOLClient) _statsGetThrottle() (int, int, int) {

	stat_requests := 0
	stat_data_received := 0
	stat_requests_per_fn_max := 0

	// calculate requests
	_pos := this.stat_last_60_pos
	for i := 0; i < throttle.ThrottleConfig.Throttle_s_requests; i++ {
		stat_requests += this.stat_last_60[_pos].stat_done
		_pos-- // take current second into account
		if _pos < 0 {
			_pos = 59
		}
	}

	// calculate data received
	_pos = this.stat_last_60_pos
	for i := 0; i < throttle.ThrottleConfig.Throttle_s_data_received; i++ {
		stat_data_received += this.stat_last_60[_pos].stat_bytes_received
		_pos-- // take current second into account
		if _pos < 0 {
			_pos = 59
		}
	}

	// calculate top function number of calls
	requests_max_per_method := make(map[string]int)
	_pos = this.stat_last_60_pos
	for i := 0; i < throttle.ThrottleConfig.Throttle_s_requests_per_fn_max; i++ {
		for k, v := range this.stat_last_60[_pos].stat_request_by_fn {
			requests_max_per_method[k] += v
		}
		_pos-- // take current second into account
		if _pos < 0 {
			_pos = 59
		}
	}
	for _, v := range requests_max_per_method {
		if v > stat_requests_per_fn_max {
			stat_requests_per_fn_max = v
		}
	}

	return stat_requests, stat_requests_per_fn_max, stat_data_received
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

func (this *SOLClient) _statsIsDead() (bool, int) {

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

	// make sure we have some requests
	if stat_requests <= 2 {
		return true, stat_requests
	}

	dead := stat_errors*5 > stat_requests
	return dead, stat_requests
}
