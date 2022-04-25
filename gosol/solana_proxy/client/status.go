package client

import (
	"fmt"
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

	// Node type
	status := ""
	_t := "Private"
	if this.is_public_node {
		_t = "Public"
	}

	// Health status
	node_stats := ""
	{
		node_health := ""
		if this.is_disabled {
			node_health = hscommon.StrMessage("Node is not healthy, recent requests failed", false)
		} else {
			node_health = hscommon.StrMessage("Node is healthy and can process requests", true)
		}
		node_health = hscommon.StrPostfixHTML(node_health, 80, " ")
		node_health += fmt.Sprintf("%d second(s) probing time\n", probe_isalive_seconds)
		node_stats += node_health
	}

	// Future status
	{
		future_status := ""
		_dead, _, _comment := this._statsIsDead()
		if _dead {
			future_status = fmt.Sprintf("New health status will be: Not Healthy ")
		} else {
			future_status = fmt.Sprintf("New health status is: Healthy ")
		}
		future_status = hscommon.StrMessage(future_status, !_dead)
		future_status = hscommon.StrPostfixHTML(future_status, 80, " ")
		future_status += _comment
		node_stats += future_status
	}

	// Throttle status
	node_stats += this.throttle.GetStatus()
	node_stats_raw := this.throttle.GetThrottleScore()

	status += "\n"
	status += "<b>" + _t + " Node Endpoint</b> " + this.endpoint + " <i>v" + this.version + "</i>"
	status += fmt.Sprintf(" ... Requests running now: %d ", this.stat_running)

	// Utilization + conserve requests
	{
		status += "Utilization: "
		util := fmt.Sprintf("%.02f%%", float64(node_stats_raw.CapacityUsed)/100.0)
		if node_stats_raw.CapacityUsed == 10000 {
			util = "<b style='color: #dd4444'>" + util + "</b>"
		} else {
			util = "<b style='color: #449944'>" + util + "</b>"
		}

		conserve_requests := ""
		if this.attr&CLIENT_CONSERVE_REQUESTS > 0 {
			conserve_requests = "<span class='tooltip' style='color: #8B4513'> (?)Conserve Requests <div>Health checks are limited for\nthis node to conserve requests.\n\nIf you're paying per-request it's good\nto enable this mode</div></span>"
		}
		status += util + conserve_requests + "\n"
	}

	status += node_stats

	// Statistics
	table := hscommon.NewTableGen("Time", "Requests", "Req/s", "Avg Time", "First Block",
		"Err JM", "Err Req", "Err Resp", "Err RResp", "Err Decode", "Sent", "Received")
	table.SetClass("tab sol")
	table.AddRow(_get_row("Last 10s", this._statsGetAggr(10), 10, "-")...)
	table.AddRow(_get_row("Last 60s", this._statsGetAggr(59), 59, "-")...)

	_fb := fmt.Sprintf("%d", this.first_available_block)
	table.AddRow(_get_row("Total", this.stat_total, int(time_running), _fb)...)
	status += table.Render()

	return status
}
