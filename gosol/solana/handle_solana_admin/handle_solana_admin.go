package handle_solana_admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/slawomir-pryczek/HSServer/handler_socket2"
	"gosol/solana_proxy"
	"math"
)

type Handle_solana_admin struct {
}

func (this *Handle_solana_admin) Initialize() {
}

func (this *Handle_solana_admin) Info() string {
	return "This plugin will allow to do proxy administration"
}

func (this *Handle_solana_admin) GetActions() []string {
	return []string{"solana_admin", "solana_admin_remove", "solana_admin_add"}
}

func (this *Handle_solana_admin) HandleAction(action string, data *handler_socket2.HSParams) string {

	err := func(s string) string {
		ret := make(map[string]interface{})
		ret["error"] = s
		tmp, _ := json.Marshal(ret)
		return string(tmp)
	}

	ok := func(s interface{}) string {
		ret := make(map[string]interface{})
		ret["result"] = s
		tmp, _ := json.Marshal(ret)
		return string(tmp)
	}

	if action == "solana_admin" {
		sch := solana_proxy.MakeScheduler()
		clients := sch.GetAll(true, true)
		clients = append(clients, sch.GetAll(false, true)...)

		out := make(map[string]interface{}, 0)
		for _, client := range clients {
			_tmp := client.GetInfo()
			out[fmt.Sprintf("client_#%d", _tmp.ID)] = _tmp
		}

		_tmp, _ := json.Marshal(out)
		data.FastReturnBNocopy(_tmp)
		return ""
	}

	if action == "solana_admin_remove" {
		id := data.GetParamI("id", -1)
		if id < 0 {
			return "Please provide client's &id="
		}

		if solana_proxy.ClientRemove(uint64(id)) {
			return ok(fmt.Sprintf("Removed client id: %d", id))
		} else {
			return err("Can't find client, nothing done")
		}
	}

	if action == "solana_admin_add" {

		id := data.GetParamI("remove_id", -1)
		node_id := uint64(0)
		if id < 0 {
			node_id = math.MaxUint64
		} else {
			node_id = uint64(id)
		}
		node := data.GetParam("node", "")
		if len(node) == 0 {
			return "Please provide &node={...JSON...} as node config, additionally you can provide &remove=node_id to replace the node with new one"
		}

		var cfg_tmp map[string]interface{}
		d := json.NewDecoder(bytes.NewReader([]byte(node)))
		d.UseNumber()
		if _err := d.Decode(&cfg_tmp); _err != nil {
			return err(_err.Error())
		}

		new_node := NodeRegisterFromConfig(cfg_tmp)
		if new_node == nil {
			return err("Error creating new node, something went wrong. Please check URL and config")
		}
		solana_proxy.ClientRemove(node_id)
		return ok(new_node.GetInfo())
	}

	return err("Something went wrong in admin module")
}
