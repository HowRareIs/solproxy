package solana_proxy

/*
import (
	"encoding/json"
	"fmt"
)

var throttle_requests = 90
var throttle_requests_fn = 33
var throttle_received = 1000 * 1000 * 95
var throttle_probe_req_seconds = 12
var throttle_probe_data_seconds = 32
var probe_isalive_seconds = 30

func (this *SOLClient) IsAlive() (bool, int) {

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
	return !dead, stat_requests
}

func (this *SOLClient) IsThrottled() ([]byte, string) {
	if !this.is_public_node {
		return nil, "Throttling disabled."
	}

	throttled_comment := ""
	req_last_10, req_last_10_fn, req_last10_received := this._getThrottleStats()
	if req_last_10 >= throttle_requests {
		throttled_comment = fmt.Sprintf("Too many requests %d/%d", req_last_10, throttle_requests)
	}
	if req_last_10_fn >= throttle_requests_fn {
		throttled_comment = fmt.Sprintf("Too many requests for single method %d/%d", req_last_10_fn, throttle_requests_fn)
	}
	if req_last10_received >= throttle_received {
		throttled_comment = fmt.Sprintf("Too much data received %d/%d", req_last10_received, throttle_received)
	}
	if len(throttled_comment) == 0 {
		return nil, "Throttling enabled. This node is not throttled."
	}

	ret := make(map[string]interface{})
	ret["error"] = "Throttled public node, please wait"
	ret["throttled"] = true
	ret["throttled_comment"] = throttled_comment
	ret["throttle_info"] = nil
	ret["throttle_timespan_seconds"] = 12

	tmp := map[string]interface{}{}
	tmp2 := map[string]interface{}{}
	tmp2["value"] = req_last_10
	tmp2["max"] = throttle_requests
	tmp2["description"] = "requests made"
	tmp["requests"] = tmp2

	tmp2 = map[string]interface{}{}
	tmp2["value"] = req_last_10_fn
	tmp2["max"] = throttle_requests_fn
	tmp2["description"] = "requests made calling single function"
	tmp["requests_fn"] = tmp2

	tmp2 = map[string]interface{}{}
	tmp2["value"] = req_last10_received
	tmp2["max"] = throttle_received
	tmp2["description"] = "bytes received"
	tmp["received"] = tmp2
	ret["throttle_info"] = tmp

	r, err := json.Marshal(ret)
	if err != nil {
		return []byte(`{"throttled":true"}`), throttled_comment
	}
	return r, throttled_comment
}

func (this *SOLClient) GetThrottledStatus() map[string]interface{} {

	ret := map[string]interface{}{}

	throttled_data, throttled_comment := this.IsThrottled()
	if !this.is_public_node {
		ret["throttled_comment"] = throttled_comment
		ret["is_throttled"] = false
		ret["used_capacity_percent"] = "0.00%"
		return ret
	}

	ret["throttled_comment"] = throttled_comment
	ret["is_throttled"] = throttled_data != nil

	req_last_10, req_last_10_fn, req_last_10_received := this._getThrottleStats()

	tmp := float64(0)
	tmp2 := float64(0)
	tmp2 = float64(req_last_10) * 100 / float64(throttle_requests)
	if tmp2 > tmp {
		tmp = tmp2
	}
	tmp2 = float64(req_last_10_fn) * 100 / float64(throttle_requests_fn)
	if tmp2 > tmp {
		tmp = tmp2
	}
	tmp2 = float64(req_last_10_received) * 100 / float64(throttle_received)
	if tmp2 > tmp {
		tmp = tmp2
	}

	ret["used_capacity"] = fmt.Sprintf("%.02f%%", tmp)
	ret["throttle_0"] = fmt.Sprintf("Throttle requests (last %d seconds): %d/%d", throttle_probe_req_seconds, req_last_10, throttle_requests)
	ret["throttle_1"] = fmt.Sprintf("Throttle requests for single method (last %d seconds): %d/%d", throttle_probe_req_seconds, req_last_10_fn, throttle_requests_fn)
	ret["throttle_2"] = fmt.Sprintf("Throttle data received (last %d seconds): %.02fMB / %.02fMB", throttle_probe_data_seconds, float64(req_last_10_received)/1000/1000, float64(throttle_received)/1000/1000)

	return ret
}
*/
