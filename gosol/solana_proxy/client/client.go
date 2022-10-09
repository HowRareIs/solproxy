package client

import (
	"gosol/solana_proxy/client/throttle"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type SOLClientAttr int

const (
	CLIENT_CONSERVE_REQUESTS SOLClientAttr = 1 << 0
)

func (this *SOLClient) SetAttr(attrs SOLClientAttr) {
	this.attr = attrs
}

func (this *SOLClient) SetPaused(paused bool, comment string) {
	this.mu.Lock()
	this.is_paused = paused
	if this.is_paused {
		this.is_paused_comment = comment
	}
	this.mu.Unlock()
}

type SOLClient struct {
	id                       uint64
	client                   *http.Client
	endpoint                 string
	is_public_node           bool
	available_block_first    int
	available_block_first_ts int64
	available_block_last     int
	available_block_last_ts  int64

	is_disabled       bool
	is_paused         bool
	is_paused_comment string

	stat_running     int
	stat_total       stat
	stat_last_60     [60]stat
	stat_last_60_pos int

	version_major int
	version_minor int
	version       string
	version_ts    int64

	mu        sync.Mutex
	serial_no uint64

	attr     SOLClientAttr
	throttle []*throttle.Throttle

	_probe_time int
	_probe_log  string

	_last_error LastError
}

type Solclientinfo struct {
	ID                       uint64
	Endpoint                 string
	Is_public_node           bool
	Available_block_first    int
	Available_block_first_ts int64
	Available_block_last     int
	Available_block_last_ts  int64
	Is_disabled              bool
	Is_throttled             bool
	Is_paused                bool

	Attr  SOLClientAttr
	Score int
}

func (this *SOLClient) GetEndpoint() string {
	this.mu.Lock()
	ret := this.endpoint
	this.mu.Unlock()

	return ret
}

func (this *SOLClient) GetInfo() *Solclientinfo {

	ret := Solclientinfo{}

	this.mu.Lock()
	ret.ID = this.id
	ret.Endpoint = this.endpoint
	ret.Is_public_node = this.is_public_node
	ret.Available_block_first = this.available_block_first
	ret.Available_block_first_ts = this.available_block_first_ts
	ret.Available_block_last = this.available_block_last
	ret.Available_block_last_ts = this.available_block_last_ts
	ret.Is_disabled = this.is_disabled
	ret.Is_paused = this.is_paused

	tmp := throttle.ThrottleGoup(this.throttle).GetThrottleScore()
	ret.Score = tmp.Score
	ret.Is_throttled = tmp.Throttled

	ret.Attr = this.attr
	this.mu.Unlock()

	return &ret
}

var new_client_id = uint64(0)

func MakeClient(endpoint string, is_public_node bool, probe_time int, max_conns int, throttle []*throttle.Throttle) *SOLClient {

	tr := &http.Transport{
		MaxIdleConns:       max_conns,
		MaxConnsPerHost:    max_conns,
		IdleConnTimeout:    10 * time.Second,
		DisableCompression: true}

	ret := SOLClient{}
	ret.client = &http.Client{Transport: tr, Timeout: 5 * time.Second}
	ret.endpoint = endpoint
	ret.is_public_node = is_public_node
	ret._probe_time = probe_time
	ret.stat_total.stat_request_by_fn = make(map[string]int)
	for i := 0; i < len(ret.stat_last_60); i++ {
		ret.stat_last_60[i].stat_request_by_fn = make(map[string]int)
	}

	ret.throttle = throttle
	ret._maintenance()

	ret.id = atomic.AddUint64(&new_client_id, 1)
	return &ret
}

func (this *SOLClient) GetThrottleLimitsLeft() (int, int, int, int) {
	this.mu.Lock()
	defer this.mu.Unlock()
	return throttle.ThrottleGoup(this.throttle).GetLimitsLeft()
}
