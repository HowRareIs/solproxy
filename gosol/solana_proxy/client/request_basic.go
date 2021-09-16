package client

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"encoding/json"
	"gosol/solana_proxy/throttle"
)

func (this *SOLClient) RequestBasic(method_param ...string) []byte {

	//	## Prepare parameters, check if the node is running
	method := method_param[0]
	params := ""
	if len(method_param) == 2 {
		params = method_param[1]
	}
	if len(method_param) > 2 {
		panic("Too many parameters!")
	}

	this.mu.Lock()
	_a, _b, _c := this._statsGetThrottle()
	is_throttled, _ := throttle.Make(this.is_public_node, _a, _b, _c).IsThrottled()
	if is_throttled != nil {
		this.mu.Unlock()
		return is_throttled
	}

	this.stat_total.stat_done++
	this.stat_last_60[this.stat_last_60_pos].stat_done++

	this.stat_total.stat_request_by_fn[method]++
	this.stat_last_60[this.stat_last_60_pos].stat_request_by_fn[method]++

	this.serial_no++
	serial_no := this.serial_no
	this.stat_running++
	this.mu.Unlock()

	// ## generate serial and prepare data
	node_type := "PRIV"
	if this.is_public_node {
		node_type = "PUB"
	}
	payload := map[string]string{}
	now := time.Now().UnixNano()
	serial := fmt.Sprintf("%s/%d/%d", node_type, serial_no, now)
	payload["jsonrpc"] = "2.0"
	payload["id"] = serial
	payload["method"] = method

	post, m_err := json.Marshal(payload)
	if m_err != nil {
		this.mu.Lock()
		this.stat_total.stat_error_json_marshal++
		this.stat_running--
		this.mu.Unlock()
		return nil
	}

	if len(params) > 0 {
		post = post[0 : len(post)-1]
		post = append(post, []byte(`,"params":`)...)
		post = append(post, []byte(params)...)
		post = append(post, '}')
	}

	req, err := http.NewRequest("POST", this.endpoint, bytes.NewBuffer(post))
	if err != nil {
		this.mu.Lock()
		this.stat_total.stat_error_req++
		this.stat_last_60[this.stat_last_60_pos].stat_error_req++
		this.stat_running--
		this.mu.Unlock()
		return nil
	}

	req.Header.Add("Content-Type", "application/json")
	resp, err := this.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		this.mu.Lock()
		this.stat_total.stat_error_resp++
		this.stat_last_60[this.stat_last_60_pos].stat_error_resp++
		this.stat_running--
		this.mu.Unlock()
		return nil
	}

	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		this.mu.Lock()
		this.stat_total.stat_error_resp_read++
		this.stat_last_60[this.stat_last_60_pos].stat_error_resp_read++
		this.stat_running--
		this.mu.Unlock()
		return nil
	}

	fmt.Println(string(resp_body))
	took := (time.Now().UnixNano() - now) / 1000
	if took < 0 {
		took = 0
	}
	this.mu.Lock()
	this.stat_total.stat_ns_total += uint64(took)
	this.stat_last_60[this.stat_last_60_pos].stat_ns_total += uint64(took)

	this.stat_total.stat_bytes_received += len(resp_body)
	this.stat_last_60[this.stat_last_60_pos].stat_bytes_received += len(resp_body)

	this.stat_total.stat_bytes_sent += len(post)
	this.stat_last_60[this.stat_last_60_pos].stat_bytes_sent += len(post)
	this.stat_running--
	this.mu.Unlock()

	if !bytes.Contains(resp_body, []byte(serial)) {
		return nil
	}
	return resp_body
}
