package solana_proxy

import (
	"github.com/slawomir-pryczek/HSServer/handler_socket2"
)

func init() {

	get_status := func() (string, string) {

		info := "This section represents individual SOLANA nodes, with number of requests and errors\n"
		info += "<b>Err JM</b> - Json Marshall error. We were unable to build JSON payload required for your request\n"
		info += "<b>Err Req</b> - Request Error. We were unable to send request to host\n"
		info += "<b>Err Resp</b> - Response Error. We were unable to get server response\n"
		info += "<b>Err RResp</b> - Response Reading Error. We were unable to read server response\n"
		info += "<b>Err Decode</b> - Json Decode Error. We were unable read received JSON\n"

		status := ""
		sh := MakeScheduler()
		for _, v := range sh.GetAll(true, true) {
			status += v.GetStatus()
		}
		for _, v := range sh.GetAll(false, true) {
			status += v.GetStatus()
		}

		return "Solana Proxy", "<pre>" + info + status + "</pre>"
	}

	handler_socket2.StatusPluginRegister(get_status)
}
