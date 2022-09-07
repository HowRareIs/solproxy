package client

import (
	"gosol/solana_proxy/client/throttle"
	"net/http"
	"sync"
	"time"
)

var probe_isalive_seconds = 30
var probe_ok_min_requests = 3

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

type SOLClientAttr int

const (
	CLIENT_CONSERVE_REQUESTS SOLClientAttr = 1 << 0
)

func (this *SOLClient) SetAttr(attrs SOLClientAttr) {
	this.attr = attrs
}

type SOLClient struct {
	client                *http.Client
	endpoint              string
	is_public_node        bool
	first_available_block int
	is_disabled           bool

	stat_running     int
	stat_total       stat
	stat_last_60     [60]stat
	stat_last_60_pos int

	version_major int
	version_minor int
	version       string

	mu        sync.Mutex
	serial_no uint64

	attr     SOLClientAttr
	throttle []*throttle.Throttle
}

type Solclientinfo struct {
	Endpoint              string
	Is_public_node        bool
	First_available_block int
	Is_disabled           bool
	Is_throttled          bool

	Attr  SOLClientAttr
	Score int
}

func (this *SOLClient) GetInfo() *Solclientinfo {

	ret := Solclientinfo{}

	this.mu.Lock()
	ret.Endpoint = this.endpoint
	ret.Is_public_node = this.is_public_node
	ret.First_available_block = this.first_available_block
	ret.Is_disabled = this.is_disabled

	tmp := throttle.ThrottleGoup(this.throttle).GetThrottleScore()
	ret.Score = tmp.Score
	ret.Is_disabled = tmp.Disabled

	ret.Attr = this.attr
	this.mu.Unlock()

	return &ret
}

func MakeClient(endpoint string, is_public_node bool, max_conns int, throttle []*throttle.Throttle) *SOLClient {

	tr := &http.Transport{
		MaxIdleConns:       max_conns,
		MaxConnsPerHost:    max_conns,
		IdleConnTimeout:    10 * time.Second,
		DisableCompression: true}

	ret := SOLClient{}
	ret.client = &http.Client{Transport: tr, Timeout: 5 * time.Second}
	ret.endpoint = endpoint
	ret.is_public_node = is_public_node
	ret.stat_total.stat_request_by_fn = make(map[string]int)
	for i := 0; i < len(ret.stat_last_60); i++ {
		ret.stat_last_60[i].stat_request_by_fn = make(map[string]int)
	}

	ret.throttle = throttle
	ret._maintenance()
	return &ret
}

func (this *SOLClient) GetThrottleLimitsLeft() (int, int, int, int) {
	this.mu.Lock()
	defer this.mu.Unlock()
	return throttle.ThrottleGoup(this.throttle).GetLimitsLeft()
}
