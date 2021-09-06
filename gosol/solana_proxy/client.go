package solana_proxy

import (
	"net/http"
	"sync"
	"time"
)

var mu sync.Mutex

type stat struct {
	stat_error_req          int
	stat_error_resp         int
	stat_error_resp_read    int
	stat_error_json_decode  int
	stat_error_json_marshal int
	stat_done               int
	stat_ns_total           uint64
}

type SOLClient struct {
	client                *http.Client
	endpoint              string
	is_public_node        bool
	first_available_block int

	stat_running     int
	stat_total       stat
	stat_last_60     [60]stat
	stat_last_60_pos int

	version_major int
	version_minor int
	version       string
}

var clients []*SOLClient

func init() {
	clients = make([]*SOLClient, 0, 10)
}

func RegisterClient(endpoint string, max_conns int, is_public_node bool) {
	tr := &http.Transport{
		MaxIdleConns:       max_conns,
		MaxConnsPerHost:    max_conns,
		IdleConnTimeout:    10 * time.Second,
		DisableCompression: true}

	ret := &SOLClient{}
	ret.client = &http.Client{Transport: tr, Timeout: 5 * time.Second}
	ret.endpoint = endpoint
	ret.is_public_node = is_public_node
	ret._maintenance()

	mu.Lock()
	clients = append(clients, ret)
	mu.Unlock()
}

func GetClient(should_be_public bool) *SOLClient {
	return GetClientB(should_be_public, -1)
}

func GetClientB(should_be_public bool, for_block int) *SOLClient {

	min := -1
	min_pos := -1
	mu.Lock()
	defer mu.Unlock()
	for num, v := range clients {
		if !v.is_public_node && should_be_public {
			continue
		}
		if for_block > -1 && for_block <= v.first_available_block {
			continue
		}

		// if the node is public it'll have limits, so get public node at the end!
		_r := v.stat_running
		if v.is_public_node {
			_r += 5000
		}

		if min == -1 || _r < min {
			min = _r
			min_pos = num
		}
	}

	if min_pos == -1 {
		return nil
	}
	ret := clients[min_pos]
	ret.stat_running++
	return ret
}

func GetMinBlocks() (int, int) {

	// a public; b private
	a, b := -1, -1
	mu.Lock()
	for _, v := range clients {
		if v.is_public_node {
			if a == -1 || v.first_available_block < a {
				a = v.first_available_block
			}
		} else {
			if b == -1 || v.first_available_block < b {
				b = v.first_available_block
			}
		}
	}
	mu.Unlock()
	return a, b
}

func (this *SOLClient) Release() {

	mu.Lock()
	this.stat_running--
	if this.stat_running < 0 {
		panic("Releasing already released client!")
	}
	mu.Unlock()
}
