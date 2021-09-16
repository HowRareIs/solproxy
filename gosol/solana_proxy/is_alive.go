package solana_proxy

/*
var probe_isalive_seconds = 30

func (this *SOLClient) IsAlive() (bool, int) {

	stat_requests := 0
	stat_errors := 0
	_pos := this.stat_last_60_pos
	for i := 0; i < probe_isalive_seconds; i++ {
		stat_requests += this.stat_last_60[_pos].stat_done
		stat_errors += this.stat_last_60[_pos].stat_error_resp

		_pos-- // take current second into account
		if _pos < 0 {
			_pos = 59
		}
	}

	// make sure we have some requests
	if stat_requests <= 2 {
		return true, stat_requests
	}

	dead := stat_errors*5 > stat_requests
	return !dead, stat_requests
}
*/
