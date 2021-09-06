package stats

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/slawomir-pryczek/handler_socket2/hscommon"
)

func GetStatus(available_actions []string, uptime int) map[string]string {

	stats_mutex.Lock()
	_connections := global_stats_connections
	_errors := global_stats_errors
	_requests := global_stats_requests
	_req_time := global_stats_req_time
	_req_time_full := global_stats_req_time_full

	_stats := make([]stats, 0, len(stats_actions))
	_stats_names := make([]string, 0, len(stats_actions))

	for k, v := range stats_actions {
		_stats = append(_stats, v.getAdjStats())
		_stats_names = append(_stats_names, k)
	}

	stats_mutex.Unlock()

	ret := map[string]string{}
	ret["_connections"] = fmt.Sprintf("%d", _connections)
	ret["_requests"] = fmt.Sprintf("%d", _requests)
	ret["_errors"] = fmt.Sprintf("%d", _errors)
	if _requests > 0 {
		ret["_req_time"] = fmt.Sprintf("%.3fms", float64(_req_time/_requests)/float64(1000))
		ret["_req_time_full"] = fmt.Sprintf("%.3fms", float64(_req_time_full/_requests)/float64(1000))
	} else {
		ret["_req_time"] = "-"
		ret["_req_time_full"] = "-"
	}
	// <<

	// Handler Status
	var _bytes_sent uint64 = 0
	var _bytes_generated uint64 = 0
	var _bytes_received uint64 = 0
	var _bytes_rec_uncompressed uint64 = 0
	var _resp_b_skipped uint64 = 0
	var _resp_skipped uint64 = 0

	table_handlers := hscommon.NewTableGen("Handler", "Calls", "AVG Req Time", "AVG Roundtrip", "Send", "S-Compression", "Received", "R-Compression", "S-Skipped", "S-Skipped Bytes")
	table_handlers.SetClass("tab")

	handlers_added := make(map[string]bool)

	for k, action_data := range _stats {
		action_type := _stats_names[k]

		_bytes_received += action_data.b_request_compressed
		_bytes_rec_uncompressed += action_data.b_request_size
		_bytes_sent += action_data.b_resp_compressed
		_bytes_generated += action_data.b_resp_size
		_resp_b_skipped += action_data.resp_b_skipped
		_resp_skipped += action_data.resp_skipped

		_r_compr := ((action_data.b_request_size + 1) * 1000) / (action_data.b_request_compressed + 1)
		_s_compr := ((action_data.b_resp_size + 1) * 1000) / (action_data.b_resp_compressed + 1)
		_rt := float64(action_data.req_time/action_data.requests) / 1000
		_rtf := float64(action_data.req_time_full/action_data.requests) / 1000

		table_handlers.AddRow(action_type, strconv.Itoa(int(action_data.requests)),
			fmt.Sprintf("%.3fms", _rt), fmt.Sprintf("%.3fms", _rtf),
			hscommon.FormatBytes(action_data.b_resp_compressed), fmt.Sprintf("%.1f%%", float64(_s_compr)/10),
			hscommon.FormatBytes(action_data.b_request_compressed), fmt.Sprintf("%.1f%%", float64(_r_compr)/10),
			fmt.Sprintf("%d", action_data.resp_skipped), hscommon.FormatBytes(action_data.resp_b_skipped))

		handlers_added[action_type] = true
	}

	for _, v := range available_actions {

		if handlers_added[v] {
			continue
		}
		table_handlers.AddRow(v, "-", "-", "-", "-", "-", "-", "-", "-", "-")
	}

	if handlers_added["server-status"] != true {
		table_handlers.AddRow("server-status", "-", "-", "-", "-", "-", "-", "-", "-", "-")
	}

	ret["handlers_table"] = table_handlers.RenderSorted(0)
	// <<

	// General info, additional
	_r_compr := ((_bytes_rec_uncompressed + 1) * 1000) / (_bytes_received + 1)
	_s_compr := ((_bytes_generated + 1) * 1000) / (_bytes_sent + 1)
	ret["_bytes_sent"] = hscommon.FormatBytes(_bytes_sent)
	ret["_bytes_received"] = hscommon.FormatBytes(_bytes_received)
	ret["_compression"] = fmt.Sprintf("%.1f%%", float64(_s_compr)/10)
	ret["_receive_compression"] = fmt.Sprintf("%.1f%%", float64(_r_compr)/10)
	ret["_resp_b_skipped"] = hscommon.FormatBytes(_resp_b_skipped)
	ret["_resp_skipped"] = fmt.Sprintf("%d", _resp_skipped)

	ret["_resp_count"] = fmt.Sprint(uint64(_requests) - uint64(_resp_skipped))
	// <<

	// #########################################################################
	//	Thread states
	_threads_states := ""
	stats_mutex.Lock()
	for _, conn_data := range status_connections {
		_threads_states += conn_data.status[0:1] + " "
	}
	stats_mutex.Unlock()
	ret["_threads_states"] = _threads_states
	// <<

	// Connections
	now := time.Now().UnixNano()
	status_items := make([]hscommon.ScoredItems, len(status_connections))
	i := 0

	stats_mutex.Lock()
	for conn_id, conn_data := range status_connections {

		var took float64 = -1
		var took_str = ""
		if conn_data.start_time != 0 {

			if conn_data.end_time == 0 {
				took = float64(now-conn_data.start_time) / float64(1000000)
			} else {
				took = float64(conn_data.end_time-conn_data.start_time) / float64(1000000)
			}

			took_str = fmt.Sprintf("%.3fms", took)
		} else {
			took_str = "??"
		}

		_conn_data := conn_data.data
		if len(_conn_data) > 60 {

			_pos := 0
			_conn_data_wbr := ""
			for _pos < len(_conn_data) {
				_end := _pos + 80
				if _end > len(_conn_data) {
					_end = len(_conn_data)
				}
				_conn_data_wbr += _conn_data[_pos:_end] + "<wbr>"
				_pos += 80
			}

			_conn_data = "<span class='tooltip'>[...] " + _conn_data[0:60] + "<div>" + _conn_data_wbr + "</div></span>"
		} else {
			_conn_data = "<span>" + _conn_data + "</span>"
		}
		_tmp := fmt.Sprintf("<span>#%d</span> - <span>[%s]</span> <span>%s</span> - <span>%s</span> %s <span>%s</span>",
			conn_id, conn_data.status, took_str,
			conn_data.action, _conn_data, conn_data.comment)
		_tmp = "<div class='thread_list'>" + _tmp + "</div>"

		status_items[i].Item = _tmp
		status_items[i].Score = int64(conn_id)
		i++
	}
	stats_mutex.Unlock()

	sort.Sort(hscommon.SIArr(status_items))

	threadlist := ""
	for _, v := range status_items {
		threadlist += v.Item
	}
	ret["threadlist"] = threadlist
	// <<

	// per second averages
	ret["_requests_s"] = fmt.Sprintf("%.2f", float64(_requests)/float64(uptime))
	ret["_connections_s"] = fmt.Sprintf("%.2f", float64(_connections)/float64(uptime))
	ret["_errors_s"] = fmt.Sprintf("%.2f", float64(_errors)/float64(uptime))

	_bytes_sent = _bytes_sent / uint64(uptime)
	_bytes_received = _bytes_received / uint64(uptime)
	ret["_bytes_sent_s"] = hscommon.FormatBytes(_bytes_sent)
	ret["_bytes_received_s"] = hscommon.FormatBytes(_bytes_received)

	// <<

	// last 5 seconds stats
	last5 := uh_get()

	ret["_requests_5s"] = fmt.Sprintf("%.2f", float64(last5.requests)/5)
	ret["_connections_5s"] = fmt.Sprintf("%.2f", float64(last5.connections)/5)
	ret["_errors_5s"] = fmt.Sprintf("%.2f", float64(last5.errors)/5)

	ret["_req_time_5s"] = fmt.Sprintf("%.3fms", float64(last5.req_time/(last5.requests+1))/float64(1000))
	ret["_req_time_full_5s"] = fmt.Sprintf("%.3fms", float64(last5.req_time_full/(last5.requests+1))/float64(1000))

	_bytes_sent = uint64(last5.b_resp_compressed / 5)
	_bytes_received = uint64(last5.b_request_compressed / 5)
	ret["_bytes_sent_5s"] = hscommon.FormatBytes(_bytes_sent)
	ret["_bytes_received_5s"] = hscommon.FormatBytes(_bytes_received)

	_s_compr = uint64(((last5.b_resp_size + 1) * 1000) / (last5.b_resp_compressed + 1))
	_r_compr = uint64(((last5.b_request_size + 1) * 1000) / (last5.b_request_compressed + 1))
	ret["_compression_5s"] = fmt.Sprintf("%.1f%%", float64(_s_compr)/10)
	ret["_receive_compression_5s"] = fmt.Sprintf("%.1f%%", float64(_r_compr)/10)

	ret["_resp_b_skipped_5s"] = hscommon.FormatBytes(last5.resp_b_skipped)
	ret["_resp_skipped_5s"] = fmt.Sprintf("%d", last5.resp_skipped)
	ret["_resp_count_5s"] = fmt.Sprint(last5.requests - last5.resp_skipped)
	// <<

	return ret

}
