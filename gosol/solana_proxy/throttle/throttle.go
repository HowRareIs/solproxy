package throttle

import (
	"encoding/json"
	"fmt"
)

type Throttle struct {
	is_public_node bool

	requests            int
	requests_per_fn_max int
	data_received       int
}

type _throttleConfig struct {
	Throttle_requests            int
	Throttle_requests_per_fn_max int
	Throttle_data_received       int

	Throttle_s_requests            int
	Throttle_s_requests_per_fn_max int
	Throttle_s_data_received       int
}

var ThrottleConfig = _throttleConfig{}

func init() {
	ThrottleConfig.Throttle_requests = 90
	ThrottleConfig.Throttle_requests_per_fn_max = 33
	ThrottleConfig.Throttle_data_received = 1000 * 1000 * 95

	ThrottleConfig.Throttle_s_requests = 12
	ThrottleConfig.Throttle_s_requests_per_fn_max = 12
	ThrottleConfig.Throttle_s_data_received = 32
}

func Make(is_public_node bool, requests, requests_per_fn_max, data_received int) Throttle {
	return Throttle{is_public_node, requests, requests_per_fn_max, data_received}
}

func (this Throttle) GetUsedCapacity() float64 {
	if !this.is_public_node {
		return 0
	}

	tmp := float64(0)
	tmp2 := float64(0)
	tmp2 = float64(this.requests) * 100 / float64(ThrottleConfig.Throttle_requests)
	if tmp2 > tmp {
		tmp = tmp2
	}
	tmp2 = float64(this.requests_per_fn_max) * 100 / float64(ThrottleConfig.Throttle_requests_per_fn_max)
	if tmp2 > tmp {
		tmp = tmp2
	}
	tmp2 = float64(this.data_received) * 100 / float64(ThrottleConfig.Throttle_data_received)
	if tmp2 > tmp {
		tmp = tmp2
	}

	_tmp := int(tmp * 1000.0)
	return float64(_tmp/100) / 10.0
}

func (this Throttle) IsThrottled() ([]byte, string) {
	if !this.is_public_node {
		return nil, "Throttling disabled."
	}

	throttled_comment := ""
	if this.requests >= ThrottleConfig.Throttle_requests {
		throttled_comment = fmt.Sprintf("Too many requests %d/%d", this.requests, ThrottleConfig.Throttle_requests)
	}
	if this.requests_per_fn_max >= ThrottleConfig.Throttle_requests_per_fn_max {
		throttled_comment = fmt.Sprintf("Too many requests for single method %d/%d", this.requests_per_fn_max, ThrottleConfig.Throttle_requests_per_fn_max)
	}
	if this.data_received >= ThrottleConfig.Throttle_data_received {
		throttled_comment = fmt.Sprintf("Too much data received %d/%d", this.data_received, ThrottleConfig.Throttle_data_received)
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
	tmp2["value"] = this.requests
	tmp2["max"] = ThrottleConfig.Throttle_requests
	tmp2["description"] = "requests made"
	tmp["requests"] = tmp2

	tmp2 = map[string]interface{}{}
	tmp2["value"] = this.requests_per_fn_max
	tmp2["max"] = ThrottleConfig.Throttle_requests_per_fn_max
	tmp2["description"] = "requests made calling single function"
	tmp["requests_fn"] = tmp2

	tmp2 = map[string]interface{}{}
	tmp2["value"] = this.data_received
	tmp2["max"] = ThrottleConfig.Throttle_data_received
	tmp2["description"] = "bytes received"
	tmp["received"] = tmp2
	ret["throttle_info"] = tmp

	r, err := json.Marshal(ret)
	if err != nil {
		return []byte(`{"throttled":true"}`), throttled_comment
	}
	return r, throttled_comment
}

func (this Throttle) GetThrottledStatus() map[string]interface{} {

	ret := map[string]interface{}{}

	throttled_data, throttled_comment := this.IsThrottled()
	if !this.is_public_node {
		ret["throttled_comment"] = throttled_comment
		ret["is_throttled"] = false
		ret["p_capacity_used"] = float64(0)
		return ret
	}

	ret["throttled_comment"] = throttled_comment
	ret["is_throttled"] = throttled_data != nil

	ret["p_capacity_used"] = this.GetUsedCapacity()
	ret["throttle_0"] = fmt.Sprintf("Throttle requests (last %d seconds): %d/%d",
		ThrottleConfig.Throttle_s_requests, this.requests, ThrottleConfig.Throttle_requests)
	ret["throttle_1"] = fmt.Sprintf("Throttle requests for single method (last %d seconds): %d/%d",
		ThrottleConfig.Throttle_s_requests_per_fn_max, this.requests_per_fn_max, ThrottleConfig.Throttle_requests_per_fn_max)
	ret["throttle_2"] = fmt.Sprintf("Throttle data received (last %d seconds): %.02fMB / %.02fMB",
		ThrottleConfig.Throttle_s_data_received, float64(this.data_received)/1000/1000, float64(ThrottleConfig.Throttle_data_received)/1000/1000)
	return ret
}
