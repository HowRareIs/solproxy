package solana_proxy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"encoding/json"
	"sync/atomic"
)

var serial_no = uint64(0)

func (this *SOLClient) RunRequest(method string) []byte {
	return this.RunRequestP(method, "")
}

func (this *SOLClient) RunRequestP(method string, params string) []byte {

	mu.Lock()
	is_throttled, _ := this.IsThrottled()
	if is_throttled != nil {
		mu.Unlock()
		return is_throttled
	}

	this.stat_total.stat_done++
	this.stat_last_60[this.stat_last_60_pos].stat_done++

	this.stat_total.stat_request_by_fn[method]++
	this.stat_last_60[this.stat_last_60_pos].stat_request_by_fn[method]++
	mu.Unlock()

	node_type := "PRIV"
	if this.is_public_node {
		node_type = "PUB"
	}
	payload := map[string]string{}
	now := time.Now().UnixNano()
	serial := fmt.Sprintf("%s/%d/%d", node_type, atomic.AddUint64(&serial_no, 1), now)
	payload["jsonrpc"] = "2.0"
	payload["id"] = serial
	payload["method"] = method

	post, m_err := json.Marshal(payload)
	if m_err != nil {
		mu.Lock()
		this.stat_total.stat_error_json_marshal++
		mu.Unlock()
		return nil
	}

	if len(params) > 0 {
		post = post[0 : len(post)-1]
		post = append(post, []byte(`,"params":`)...)
		post = append(post, []byte(params)...)
		post = append(post, '}')
		fmt.Println(">>>", string(post))
	}

	req, err := http.NewRequest("POST", this.endpoint, bytes.NewBuffer(post))
	if err != nil {
		mu.Lock()
		this.stat_total.stat_error_req++
		this.stat_last_60[this.stat_last_60_pos].stat_error_req++
		mu.Unlock()
		return nil
	}

	req.Header.Add("Content-Type", "application/json")
	resp, err := this.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		mu.Lock()
		this.stat_total.stat_error_resp++
		this.stat_last_60[this.stat_last_60_pos].stat_error_resp++
		mu.Unlock()
		return nil
	}

	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		mu.Lock()
		this.stat_total.stat_error_resp_read++
		this.stat_last_60[this.stat_last_60_pos].stat_error_resp_read++
		mu.Unlock()
		return nil
	}

	fmt.Println(string(resp_body))
	took := (time.Now().UnixNano() - now) / 1000
	if took < 0 {
		took = 0
	}
	mu.Lock()
	this.stat_total.stat_ns_total += uint64(took)
	this.stat_last_60[this.stat_last_60_pos].stat_ns_total += uint64(took)

	this.stat_total.stat_bytes_received += len(resp_body)
	this.stat_last_60[this.stat_last_60_pos].stat_bytes_received += len(resp_body)

	this.stat_total.stat_bytes_sent += len(post)
	this.stat_last_60[this.stat_last_60_pos].stat_bytes_sent += len(post)
	mu.Unlock()

	if !bytes.Contains(resp_body, []byte(serial)) {
		return nil
	}
	return resp_body
}
