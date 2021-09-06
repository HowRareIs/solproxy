package solana_proxy

import (
	"fmt"
	"time"
)

func (this *SOLClient) _maintenance() {

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

	_maint_stat := func(now int64) {
		_p := int(now % 60)
		this.stat_last_60_pos = _p
		this.stat_last_60[_p].stat_done = 0
		this.stat_last_60[_p].stat_error_json_decode = 0
		this.stat_last_60[_p].stat_error_json_marshal = 0
		this.stat_last_60[_p].stat_error_req = 0
		this.stat_last_60[_p].stat_error_resp = 0
		this.stat_last_60[_p].stat_error_resp_read = 0
		this.stat_last_60[_p].stat_ns_total = 0
	}
	_maint := func(now int64) {
		if !this.is_public_node {
			_b, _ok := this.GetFirstAvailableBlock()
			if !_ok {
				return
			}

			mu.Lock()
			this.first_available_block = _b
			_maint_stat(now)
			should_update_version := this.version_major == 0
			mu.Unlock()
			if should_update_version {
				_update_version()
			}
			return
		}

		mu.Lock()
		_maint_stat(now)
		should_update_version := this.version_major == 0
		mu.Unlock()
		if should_update_version {
			_update_version()
		}
	}

	_maint(time.Now().Unix())
	go func() {

		now := time.Now().Unix()
		for {
			time.Sleep(250 * time.Millisecond)
			_t := time.Now().Unix()
			if now >= _t {
				continue
			}

			now = _t
			_maint(now)
		}
	}()
}
