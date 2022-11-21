package throttle

import (
	"fmt"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/hscommon"
	"strings"
)

/* This has to hold mutex externally */
func (this *Throttle) GetStatus() string {

	_progress := func(p int) string {
		if p > 100 {
			p = 100
		}
		p = p / 10

		ret := strings.Repeat("◆", p)
		if p < 10 && 10-p > 0 {
			ret = ret + strings.Repeat("◇", 10-p)
		}
		return ret
	}

	status := "<span style='color: #449944; font-family: monospace'> <b>⬤</b> Throttling disabled (##layout##) ⏵︎⏵︎⏵︎</span>"
	if (len(this.limiters) > 0) && this.status_throttled {
		status = "<span style='color: #dd4444; font-family: monospace'> <b>⮿</b> Throttling group (##layout##), exhausted</span>"
	}
	if (len(this.limiters) > 0) && !this.status_throttled {
		status = "<span style='color: #449944; font-family: monospace'><b>⬤</b> Throttling group (##layout##)</span>"
	}
	status = strings.Replace(status, "##layout##", fmt.Sprintf("%dx%ds", len(this.stats), this.stats_window_size_seconds), 1)

	status = hscommon.StrPostfixHTML(status, 80, " ")
	status += fmt.Sprintf("Group Score: %d (Modifier: %d)\n", this.status_score, this.score_modifier)

	for k, _ := range this.limiters {
		v := &this.limiters[k]
		_type := "requests"
		if v.t == L_REQUESTS_PER_FN {
			_type = "requests for single function"
		}
		if v.t == L_DATA_RECEIVED {
			_type = "bytes received"
		}

		_s := v.in_time_windows * this.stats_window_size_seconds
		thr_status := fmt.Sprintf("Throtting #%d: %d second(s), maximum %d %s", k, _s, v.maximum, _type)
		thr_status = hscommon.StrPostfix(thr_status, 80, " ")

		if len(thr_status) < 80 {
			thr_status += strings.Repeat(" ", 80-len(thr_status))
		}

		_amt, _perc := this._getThrottleStatus(v)
		color := "#000000"
		symbol := "⬤"
		if _perc >= 10000 {
			color = "#dd4444"
			symbol = "⮿"
		}

		thr_status += _progress(_perc/100) + "\t"
		thr_status += fmt.Sprintf("<span style='color: %s; font-family: monospace'><b>%s</b> %d/%d (%.01f%%) <i>+%d</i></span>\n",
			color, symbol, _amt, v.maximum, float64(_perc)/100.0, _perc)
		status += thr_status
	}

	return status
}

func (this *Throttle) GetLimitsLeft() (int, int, int, int) {
	tmp := this.GetThrottleScore()
	if tmp.Throttled {
		return 0, 0, 0, 10000
	}

	capacity_used := 0
	left_reqs := 1024 * 1024 * 1024
	left_reqs_fn := 1024 * 1024 * 1024
	left_data_rec := 1024 * 1024 * 1024

	for k, _ := range this.limiters {
		v := &this.limiters[k]
		amt_used, cap_used := this._getThrottleStatus(v)
		amt_left := v.maximum - amt_used
		if amt_left < 0 {
			amt_left = 0
		}

		if v.t == L_REQUESTS && amt_left < left_reqs {
			left_reqs = amt_left
		}
		if v.t == L_REQUESTS_PER_FN && amt_left < left_reqs_fn {
			left_reqs_fn = amt_left
		}
		if v.t == L_DATA_RECEIVED && amt_left < left_data_rec {
			left_data_rec = amt_left
		}
		if cap_used > capacity_used {
			capacity_used = cap_used
		}
	}

	if capacity_used > 10000 {
		capacity_used = 10000
	}

	return left_reqs, left_reqs_fn, left_data_rec, capacity_used
}
