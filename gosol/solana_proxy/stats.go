package solana_proxy

import (
	"fmt"
	"time"

	"github.com/slawomir-pryczek/handler_socket2"
	"github.com/slawomir-pryczek/handler_socket2/hscommon"
)

func (this *SOLClient) _statsAggr(seconds int) stat {

	s := stat{}
	_pos := this.stat_last_60_pos
	for i := 0; i < seconds; i++ {
		_pos--
		if _pos < 0 {
			_pos = 59
		}

		_tmp := this.stat_last_60[_pos%60]
		s.stat_done += _tmp.stat_done
		s.stat_error_json_decode += _tmp.stat_error_json_decode
		s.stat_error_json_marshal += _tmp.stat_error_json_marshal
		s.stat_error_req += _tmp.stat_error_req
		s.stat_error_resp += _tmp.stat_error_resp
		s.stat_error_resp_read += _tmp.stat_error_resp_read
		s.stat_ns_total += _tmp.stat_ns_total
	}

	return s
}

func init() {

	start_time := time.Now().Unix()

	handler_socket2.StatusPluginRegister(func() (string, string) {

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
			return _r
		}

		time_running := time.Now().Unix() - start_time
		mu.Lock()

		status := ""
		for _, v := range clients {

			table := hscommon.NewTableGen("Time", "Requests", "Req/s", "Avg Time", "First Block",
				"Err JM", "Err Req", "Err Resp", "Err RResp", "Err Decode")
			table.SetClass("tab sol")

			_t := "Private"
			if v.is_public_node {
				_t = "Public"
			}

			status += "<br>"
			status += _t + " Node Endpoint: " + v.endpoint + " <i>v" + v.version + "</i> ... Requests running now: " + fmt.Sprintf("%d", v.stat_running)
			table.AddRow(_get_row("Last 10s", v._statsAggr(10), 10, "-")...)
			table.AddRow(_get_row("Last 60s", v._statsAggr(59), 59, "-")...)

			_fb := fmt.Sprintf("%d", v.first_available_block)
			table.AddRow(_get_row("Total", v.stat_total, int(time_running), _fb)...)
			status += table.Render()
		}
		mu.Unlock()

		info := "<pre>This section represents individual SOLANA nodes, with number of requests and errors\n"
		info += "<b>Err JM</b> - Json Marshall error. We were unable to build JSON payload required for your request\n"
		info += "<b>Err Req</b> - Request Error. We were unable to send request to host\n"
		info += "<b>Err Resp</b> - Response Error. We were unable to get server response\n"
		info += "<b>Err RResp</b> - Response Reading Error. We were unable to read server response\n"
		info += "<b>Err Decode</b> - Json Decode Error. We were unable read received JSON\n"
		return "Solana Proxy", info + status
	})
}
