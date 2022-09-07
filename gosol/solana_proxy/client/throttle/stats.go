package throttle

type stat struct {
	stat_requests      int
	stat_request_by_fn map[string]int
	stat_data_received int
}

func (this *Throttle) OnRequest(function_name string) bool {
	if this.status_disabled {
		return false
	}

	to_mod := &this.stats[this.stats_pos]
	to_mod.stat_requests++
	to_mod.stat_request_by_fn[function_name]++

	// update statistics
	tmp := this._getThrottleScore()
	this.status_disabled = tmp.Disabled
	this.status_score = tmp.Score
	return true
}

func (this *Throttle) OnReceive(data_bytes int) {
	to_mod := &this.stats[this.stats_pos]
	to_mod.stat_data_received += data_bytes
}

func (this *Throttle) OnMaintenance(ts int) {
	new_pos := (ts / this.stats_window_size_seconds) % len(this.stats)
	if new_pos == this.stats_pos {
		return
	}

	this.stats_pos = new_pos
	this.stats[this.stats_pos].stat_requests = 0
	this.stats[this.stats_pos].stat_request_by_fn = make(map[string]int)
	this.stats[this.stats_pos].stat_data_received = 0

	// update statistics, as data is changing
	tmp := this._getThrottleScore()
	this.status_disabled = tmp.Disabled
	this.status_score = tmp.Score
	this.status_capacity_used = tmp.CapacityUsed
}

// Get throttle status, first return parameter is amount, second amount used 0-10000
func (this *Throttle) _getThrottleStatus(l *Limiter) (int, int) {

	amt := 0
	pos := this.stats_pos

	if l.t == L_REQUESTS {
		for i := 0; i < l.in_time_windows; i++ {
			amt += this.stats[pos].stat_requests
			pos--
			if pos < 0 {
				pos = len(this.stats) - 1
			}
		}
	}

	if l.t == L_DATA_RECEIVED {
		for i := 0; i < l.in_time_windows; i++ {
			amt += this.stats[pos].stat_data_received
			pos--
			if pos < 0 {
				pos = len(this.stats) - 1
			}
		}
	}

	if l.t == L_REQUESTS_PER_FN {
		tmp := make(map[string]int)

		for i := 0; i < l.in_time_windows; i++ {
			for k, v := range this.stats[pos].stat_request_by_fn {
				tmp[k] += v
			}
			pos--
			if pos < 0 {
				pos = len(this.stats) - 1
			}
		}

		for _, v := range tmp {
			if amt < v {
				amt = v
			}
		}
	}

	percentage_used := 0
	if amt >= l.maximum {
		percentage_used = 10000
	} else {
		percentage_used = int((float64(amt) * 10000) / float64(l.maximum))
	}

	return amt, percentage_used
}

// Get Score 0-10000
type ThrottleScore struct {
	Score        int
	Disabled     bool
	CapacityUsed int
}

func (this *Throttle) _getThrottleScore() ThrottleScore {

	// for non-throttled nodes get score based on last 10 seconds of data
	// every request is worth 1 point
	// every 10kb of data is worth 1 point
	if len(this.limiters) == 0 {
		pos := this.stats_pos
		score := 0
		for i := 0; i < 10; i++ {
			score += this.stats[pos].stat_requests + (this.stats[pos].stat_data_received / 10000)
			pos--
			if pos < 0 {
				pos = len(this.stats) - 1
			}
		}
		score += this.score_modifier
		return ThrottleScore{score, false, 0}
	}

	score := 0
	disabled := false
	for k, _ := range this.limiters {
		_, tmp := this._getThrottleStatus(&this.limiters[k])
		if tmp > score {
			score = tmp
		}
	}
	capacity_used := score
	if score >= 10000 {
		disabled = true
	}

	score += this.score_modifier
	return ThrottleScore{score, disabled, capacity_used}
}

func (this *Throttle) GetThrottleScore() ThrottleScore {
	return ThrottleScore{this.status_score, this.status_disabled, this.status_capacity_used}
}
