package handle_solana_raw

import (
	"encoding/json"
	"gosol/solana_proxy"
	"gosol/solana_proxy/client"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/slawomir-pryczek/handler_socket2"
)

func _passthrough_err(err string) []byte {
	out := make(map[string]interface{}, 0)
	out["message"] = err
	out["code"] = 111
	out["proxy_error"] = true
	b, e := json.Marshal(out)
	if e != nil {
		b = []byte("Unknown error")
	}
	return []byte("{\"error\":\"" + string(b) + "\"}")
}

func init() {

	handler_socket2.HTTPPluginRegister(func(w http.ResponseWriter, r *http.Request) bool {

		is_sol_rpc := strings.EqualFold("application/json", r.Header.Get("Content-Type"))
		if !is_sol_rpc {
			return false
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return false
		}
		r.Body.Close()

		is_sol_rpc = false
		for i := 0; i < len(body); i++ {
			if body[i] == '{' {
				is_sol_rpc = true
				break
			}
			if body[i] == '\n' || body[i] == '\r' || body[i] == ' ' {
				continue
			}
			break // we couldn't find JSON bracket, so it's not SOL RPC
		}
		if !is_sol_rpc {
			return false
		}

		sch := solana_proxy.MakeScheduler()
		clients := sch.GetAll(true, false)
		if len(clients) == 0 {
			w.Write(_passthrough_err("Can't find any client"))
			return true
		}

		errors := 0
		for _, cl := range clients {
			resp_type, resp_data := cl.RequestForward(body)
			if resp_type == client.FORWARD_OK {
				w.Write(resp_data)
				return true
			}

			if resp_type == client.FORWARD_ERROR {
				errors++
				if errors >= 2 {
					w.Write(_passthrough_err("Request failed (e)"))
					return true
				}
			}
		}

		w.Write(_passthrough_err("Request failed"))
		return true
	})
}
