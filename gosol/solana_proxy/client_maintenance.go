package solana_proxy

import (
	"fmt"
	"time"
)

func (this *SOLClient) _maintenance() {

	_maint_stat := func(now int64) {
		mu.Lock()
		_p := int(now % 60)
		this.stat_last_60_pos = _p
		this.stat_last_60[_p].stat_done = 0
		this.stat_last_60[_p].stat_error_json_decode = 0
		this.stat_last_60[_p].stat_error_json_marshal = 0
		this.stat_last_60[_p].stat_error_req = 0
		this.stat_last_60[_p].stat_error_resp = 0
		this.stat_last_60[_p].stat_error_resp_read = 0
		this.stat_last_60[_p].stat_ns_total = 0

		this.stat_last_60[_p].stat_request_by_fn = make(map[string]int)
		this.stat_last_60[_p].stat_bytes_received = 0
		this.stat_last_60[_p].stat_bytes_sent = 0

		_d, _requests_done := this.IsAlive()
		this.is_disabled = _d
		if _requests_done < 5 {
			go func() {
				this.GetVersion() // run a request to check if the node is alive
			}()
		}
		mu.Unlock()
	}

	_update_version := func() {
		_a, _b, _c, ok := this.GetVersion()
		if !ok {
			fmt.Println("Can't get version for: ", this.endpoint)
			return
		}
		mu.Lock()
		this.version_major, this.version_minor, this.version = _a, _b, _c
		mu.Unlock()
	}
	_maint := func() {
		if !this.is_public_node {
			_b, _ok := this.GetFirstAvailableBlock()
			if !_ok {
				return
			}

			mu.Lock()
			this.first_available_block = _b
			should_update_version := this.version_major == 0
			mu.Unlock()
			if should_update_version {
				_update_version()
			}
			return
		}

		mu.Lock()
		should_update_version := this.version_major == 0
		mu.Unlock()
		if should_update_version {
			_update_version()
		}
	}

	_maint()
	go func() {
		for {
			now := time.Now().Unix()
			time.Sleep(500 * time.Millisecond)
			_t := time.Now().Unix()
			if now >= _t {
				continue
			}

			now = _t
			_maint()
		}
	}()

	_maint_stat(time.Now().Unix())
	go func() {
		for {
			now := time.Now().Unix()
			time.Sleep(250 * time.Millisecond)
			_t := time.Now().Unix()
			if now >= _t {
				continue
			}

			now = _t
			_maint_stat(now)
		}
	}()
}
