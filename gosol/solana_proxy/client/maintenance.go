package client

import (
	"fmt"
	"gosol/solana_proxy/client/throttle"
	"time"
)

func (this *SOLClient) _maintenance() {

	_maint_stat := func(now int64) {
		this.mu.Lock()

		throttle.ThrottleGoup(this.throttle).OnMaintenance(int(now))

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

		_d, _requests_done, _ := this._statsIsDead()
		this.is_disabled = _d

		should_update := _requests_done < 5 && this.attr&CLIENT_CONSERVE_REQUESTS == 0
		should_update = should_update || _requests_done < 1 && this.attr&CLIENT_CONSERVE_REQUESTS == 1
		if should_update {
			go func() {
				this.GetVersion() // run a request to check if the node is alive
			}()
		}
		this.mu.Unlock()
	}

	_update_version := func() {
		_a, _b, _c, ok := this.GetVersion()
		if ok != R_OK {
			fmt.Println("Can't get version for: ", this.endpoint)
			return
		}
		this.mu.Lock()
		this.version_major, this.version_minor, this.version = _a, _b, _c
		this.mu.Unlock()
	}

	_update_first_block := func() {
		_b, _ok := this.GetFirstAvailableBlock()
		if _ok != R_OK {
			fmt.Println("Can't get first block for: ", this.endpoint)
			return
		}

		this.mu.Lock()
		this.first_available_block = _b
		this.mu.Unlock()
	}

	// run first update, get all data required for the node to work!
	_update_version()
	_update_first_block()
	go func() {
		for {
			now := time.Now().Unix()
			time.Sleep(500 * time.Millisecond)
			_t := time.Now().Unix()
			if now >= _t {
				continue
			}

			// update version and first block
			now = _t

			// conserve requests, don't probe as often
			if this.attr&CLIENT_CONSERVE_REQUESTS > 0 {
				if now%120 == 0 {
					_update_version()
				}
				if now%120 == 60 {
					_update_first_block()
				}
				continue
			}

			if (now%2 == 0 && !this.is_public_node) || now%20 == 0 {
				_update_version()
			}
			if (now%2 == 1 && !this.is_public_node) || now%20 == 10 {
				_update_first_block()
			}
		}
	}()

	_maint_stat(time.Now().Unix())
	go func() {
		for {
			now := time.Now().Unix()
			time.Sleep(200 * time.Millisecond)
			_t := time.Now().Unix()
			if now >= _t {
				continue
			}

			now = _t
			_maint_stat(now)
		}
	}()
}
