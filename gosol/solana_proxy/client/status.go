package client

import (
	"fmt"
	"gosol/solana_proxy/throttle"
	"strings"
	"time"

	"github.com/slawomir-pryczek/handler_socket2/hscommon"
)

var start_time = int64(0)

func init() {
	start_time = time.Now().Unix()
}

func (this *SOLClient) GetStatus() string {

	_get_row := func(label string, s stat, time_running int, _addl ...string) []string {

		_req := fmt.Sprintf("%d", s.stat_done)
		_req_s := fmt.Sprintf("%.02f", float64(s.stat_done)/float64(time_running))
		_req_avg := fmt.Sprintf("%.02f ms", (float64(s.stat_ns_total)/float64(s.stat_done))/1000.0)

		_r := make([]string, 0, 10)
		_r = append(_r, label, _req, _req_s, _req_avg)
		_r = append(_r, _addl...)

		_r = append(_r, fmt.Sprintf("%d", s.stat_error_json_marshal))
		_r = append(_r, fmt.Sprintf("%d", s.stat_error_req))
		_r = append(_r, fmt.Sprintf("%d", s.stat_error_resp))
		_r = append(_r, fmt.Sprintf("%d", s.stat_error_resp_read))
		_r = append(_r, fmt.Sprintf("%d", s.stat_error_json_decode))

		_r = append(_r, fmt.Sprintf("%.02fMB", float64(s.stat_bytes_sent)/1000/1000))
		_r = append(_r, fmt.Sprintf("%.02fMB", float64(s.stat_bytes_received)/1000/1000))
		return _r
	}

	time_running := time.Now().Unix() - start_time

	this.mu.Lock()
	defer this.mu.Unlock()

	status := ""
	_a, _b, _c := this._statsGetThrottle()
	thr := throttle.Make(this.is_public_node, _a, _b, _c)

	_t := "Private"
	if this.is_public_node {
		_t = "Public"
	}

	_tmp := thr.GetThrottledStatus()
	color := "#44aa44"
	if _tmp["is_throttled"].(bool) {
		color = "#aa4444"
	}

	node_stats := "<b style='background: #FF7777!important'>Broken / Disabled!</b>"
	if !this.is_disabled {
		node_stats = "<b style='background: #77FF77!important'>Running</b>"
	}

	__t, __t2 := this._statsIsDead()
	node_stats = fmt.Sprintf("Node status: %s, Based on current stats (%d seconds) next alive status is: <b>%v</b> (using %d requests)<br>",
		node_stats, probe_isalive_seconds, !__t, __t2)

	throttle_stats := fmt.Sprintf("Throttle settings: <b style='color:%s'>%s</b>\n", color, _tmp["throttled_comment"])
	for k, v := range _tmp {
		if strings.Index(k, "throttle_") == -1 {
			continue
		}
		throttle_stats += fmt.Sprintf("<b>%s</b>: %s\n", k, v)
	}

	table := hscommon.NewTableGen("Time", "Requests", "Req/s", "Avg Time", "First Block",
		"Err JM", "Err Req", "Err Resp", "Err RResp", "Err Decode", "Sent", "Received")
	table.SetClass("tab sol")

	status += "\n"
	status += _t + " Node Endpoint: " + this.endpoint + " <i>v" + this.version + "</i>"
	status += fmt.Sprintf(" ... Requests running now: %d ", this.stat_running)
	status += fmt.Sprintf("Utilization: %.02f%%\n", thr.GetUsedCapacity())

	status += node_stats
	status += throttle_stats

	table.AddRow(_get_row("Last 10s", this._statsGetAggr(10), 10, "-")...)
	table.AddRow(_get_row("Last 60s", this._statsGetAggr(59), 59, "-")...)

	_fb := fmt.Sprintf("%d", this.first_available_block)
	table.AddRow(_get_row("Total", this.stat_total, int(time_running), _fb)...)
	status += table.Render()

	return status
}
