package handle_solana_raw

import (
	"bytes"
	"encoding/json"

	"gosol/solana_proxy"

	"github.com/slawomir-pryczek/handler_socket2"
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

	// Froce single node
	force_public := data.GetParamI("public", 0) == 1
	force_private := data.GetParamI("private", 0) == 1
	if force_public || force_private {
		client := solana_proxy.GetClient(force_public)
		if client == nil {
			return `{"error":"can't find appropriate client"}`
		}
		ret := client.RunRequestP(method, params)
		defer client.Release()
		if ret == nil {
			return `{"error":"client error!"}`
		}
		data.FastReturnBNocopy(ret)
		return ""
	}

	// #######
	// Try private client
	client := solana_proxy.GetClient(false)
	if client == nil {
		return `{"error":"can't find appropriate client"}`
	}
	ret := client.RunRequestP(method, params)
	defer client.Release()
	if ret != nil && is_req_ok(ret) {
		data.FastReturnBNocopy(ret)
		return ""
	}

	// #######
	// Resort to public if private failed
	client2 := solana_proxy.GetClient(true)
	if client2 != nil {
		ret = client2.RunRequestP(method, params)
		client2.Release()
	}
	if ret == nil {
		return `{"error":"unknown issue"}`
	}
	data.FastReturnBNocopy(ret)
	return ""

}
