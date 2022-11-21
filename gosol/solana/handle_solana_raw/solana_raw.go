package handle_solana_raw

import (
	"bytes"
	"encoding/json"
	"gosol/solana_proxy"
	"gosol/solana_proxy/client"

	"github.com/slawomir-pryczek/HSServer/handler_socket2"
)

type Handle_solana_raw struct {
}

func (this *Handle_solana_raw) Initialize() {
}

func (this *Handle_solana_raw) Info() string {
	return "This plugin will allow to issue raw solana requests"
}

func (this *Handle_solana_raw) GetActions() []string {
	return []string{"solanaRaw"}
}

func (this *Handle_solana_raw) HandleAction(action string, data *handler_socket2.HSParams) string {

	// get first client!
	sch := solana_proxy.MakeScheduler()
	if data.GetParamI("public", 0) == 1 {
		sch.ForcePublic(true)
	}
	if data.GetParamI("private", 0) == 1 {
		sch.ForcePrivate(true)
	}
	cl := sch.GetAnyClient()
	if cl == nil {
		return `{"error":"can't find appropriate client"}`
	}

	// run the request
	is_req_ok := func(data []byte) bool {
		v := make(map[string]interface{})
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		dec.Decode(&v)

		switch v["result"].(type) {
		case nil:
			return false
		}
		return true
	}

	method := data.GetParam("method", "")
	params := data.GetParam("params", "")
	if len(method) == 0 {
		return `{"error":"provide transaction &method=getConfirmedBlock and optionally &amp;params=[94435095] add &public=1 if you want to force the request to be run on public node"}`
	}

	// Try first client (private by default)
	ret, result := cl.RequestBasic(method, params)
	if ret != nil && result == client.R_OK && is_req_ok(ret) {
		data.FastReturnBNocopy(ret)
		return ""
	}

	// Try public client, if private failed
	cl = sch.GetPublicClient()
	if cl != nil {
		ret, result = cl.RequestBasic(method, params)
	}
	if ret != nil && result == client.R_OK && is_req_ok(ret) {
		data.FastReturnBNocopy(ret)
		return ""
	}

	// last result, try anything which is not throttled!
	for result == client.R_THROTTLED {
		cl = sch.GetAnyClient()
		if cl == nil {
			break
		}
		ret, result = cl.RequestBasic(method, params)
	}
	if ret != nil && result == client.R_OK && is_req_ok(ret) {
		data.FastReturnBNocopy(ret)
		return ""
	}

	// return error, if we were not able to process the request correctly
	if ret != nil {
		data.FastReturnBNocopy(ret)
		return ""
	}
	return `{"error":"unknown issue"}`
}
