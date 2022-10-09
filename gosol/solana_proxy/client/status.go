package client

import (
	"fmt"
	node_status "gosol/solana_proxy/client/status"
	"gosol/solana_proxy/client/throttle"
	"html"
	"time"

	"github.com/slawomir-pryczek/handler_socket2/hscommon"
)

var start_time = int64(0)

func init() {
	start_time = time.Now().Unix()
}

func (this *SOLClient) GetStatus() string {

	status_throttle := throttle.ThrottleGoup(this.throttle).GetThrottleScore()
	out, status_description := node_status.Create(this.is_paused, status_throttle.Throttled, this.is_disabled)

	// Node name and status description
	{
		header := ""
		_t := "Private"
		if this.is_public_node {
			_t = "Public"
		}
		_e := this.endpoint
		_util := fmt.Sprintf("%.02f%%", float64(status_throttle.CapacityUsed)/100.0)
		header += fmt.Sprintf("<b>%s Node #%d</b>, Score: %d, Utilization: %s, %s\n", _t, this.id, status_throttle.Score, _util, _e)
		header += status_description
		header += "\n"
		header += this._probe_log
		out.SetHeader(header)
	}

	// Add basic badges
	if len(this.version) > 0 {
		out.AddBadge("Version: "+this.version, node_status.Gray, "Version number was updated on: "+time.UnixMilli(this.version_ts).Format("2006-01-02 15:04:05"))
	}
	out.AddBadge(fmt.Sprintf("%d Requests Running", this.stat_running), node_status.Gray, "Number of requests currently being processed.")
	if this._probe_time >= 10 {
		out.AddBadge("Conserve Requests", node_status.Green, "Health checks are limited for\nthis node to conserve requests.\n\nIf you're paying per-request\nit's good to enable this mode.")
	}

	// show last error if we have any
	if this._last_error.counter > 0 {
		last_error_header, last_error_details := this._last_error.Info()
		_comment := html.EscapeString(last_error_header) + "\n" + html.EscapeString(last_error_details)
		out.AddBadge(fmt.Sprintf("Has Errors: %d", this._last_error.counter), node_status.Orange, _comment)
	}

	// Next health badge
	{
		_dead, r, e, _comment := this._statsIsDead()
		_comment = "Node status which will be applied during the next update:\n" + _comment
		if _dead {
			out.AddBadge(fmt.Sprintf("Predicted Not Healthy (%dR/%dE)", r, e), node_status.Red, _comment)
		} else {
			out.AddBadge(fmt.Sprintf("Predicted Healthy (%dR/%dE)", r, e), node_status.Green, _comment)
		}
	}

	// Paused status
	{
		if this.is_paused {
			_p := "Node is paused"
			if len(this.is_paused_comment) > 0 {
				_p += ", reason:\n" + this.is_paused_comment
			} else {
				_p += ", no additional info present"
			}
			out.AddBadge("Paused", node_status.Gray, _p)
		}
	}

	// Generate content (throttle settings)
	{
		content := ""
		for _, throttle := range this.throttle {
			content += throttle.GetStatus()
		}
		out.AddContent(content)
	}

	// Requests statistics
	{
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
		// Statistics
		table := hscommon.NewTableGen("Time", "Requests", "Req/s", "Avg Time", "First Block", "Last Block",
			"Err JM", "Err Req", "Err Resp", "Err RResp", "Err Decode", "Sent", "Received")
		table.SetClass("tab sol")
		table.AddRow(_get_row("Last 10s", this._statsGetAggr(10), 10, "-", "-")...)
		table.AddRow(_get_row("Last 60s", this._statsGetAggr(60), 60, "-", "-")...)

		_fb := fmt.Sprintf("%d", this.available_block_first)
		_lb := fmt.Sprintf("%d", this.available_block_last)

		time_running := time.Now().Unix() - start_time
		table.AddRow(_get_row("Total", this.stat_total, int(time_running), _fb, _lb)...)
		out.AddContent(table.Render())
	}

	return "\n" + out.Render()
}
