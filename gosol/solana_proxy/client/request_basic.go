package client

import (
	"bytes"
	"fmt"
	"gosol/solana_proxy/client/throttle"
	"io/ioutil"
	"net/http"
	"time"

	"encoding/json"
)

type ResponseType uint8

const (
	R_OK        ResponseType = 0
	R_ERROR                  = 1
	R_THROTTLED              = 2
	R_REDO                   = 50
)

const FORWARD_OK = 0
const FORWARD_ERROR = 1
const FORWARD_THROTTLE = 10

func (this *SOLClient) RequestForward(body []byte) (ResponseType, []byte) {

	type method_def struct {
		Method string `json:"method"`
	}
	tmp := &method_def{}
	if json.Unmarshal(body, tmp) != nil {
		return R_ERROR, []byte("{\"error\":\"json unmarshall error\"}")
	}
	method := tmp.Method
	if len(method) == 0 {
		method = "?"
	}

	this.mu.Lock()

	// THROTTLE BLOCK! Check if we're not throttled
	if throttle.ThrottleGoup(this.throttle).GetThrottleScore().Throttled {
		this.mu.Unlock()
		return R_THROTTLED, nil
	}
	throttle.ThrottleGoup(this.throttle).OnRequest(method)
	// <<

	this.stat_total.stat_done++
	this.stat_last_60[this.stat_last_60_pos].stat_done++

	this.stat_total.stat_request_by_fn[method]++
	this.stat_last_60[this.stat_last_60_pos].stat_request_by_fn[method]++

	this.serial_no++
	this.stat_running++
	this.mu.Unlock()

	now := time.Now().UnixNano()
	resp_body := this._docall(now, body)
	if resp_body == nil {
		return R_ERROR, []byte("{\"error\":\"nil response from docall\"}")
	}
	return R_OK, resp_body
}

func (this *SOLClient) RequestBasic(method_param ...string) ([]byte, ResponseType) {

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
	// THROTTLE BLOCK! Check if we're not throttled
	if throttle.ThrottleGoup(this.throttle).GetThrottleScore().Throttled {
		this.mu.Unlock()
		return nil, R_THROTTLED
	}
	throttle.ThrottleGoup(this.throttle).OnRequest(method)

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
		return nil, R_ERROR
	}

	if len(params) > 0 {
		post = post[0 : len(post)-1]
		post = append(post, []byte(`,"params":`)...)
		post = append(post, []byte(params)...)
		post = append(post, '}')
	}

	resp_body := this._docall(now, post)
	if !bytes.Contains(resp_body, []byte(serial)) {

		fmt.Println(">ERROR IN RESPONSE>", string(resp_body))
		this.mu.Lock()
		this.stat_total.stat_error_resp++
		this.stat_last_60[this.stat_last_60_pos].stat_error_resp++
		this.mu.Unlock()
		return nil, R_ERROR
	}
	return resp_body, R_OK
}

func (this *SOLClient) _docall(ts_started int64, post []byte) []byte {
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

	took := (time.Now().UnixNano() - ts_started) / 1000
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
	throttle.ThrottleGoup(this.throttle).OnReceive(len(resp_body))
	this.mu.Unlock()
	return resp_body
}
