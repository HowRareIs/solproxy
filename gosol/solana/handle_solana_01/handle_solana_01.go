package handle_solana_01

import (
	"encoding/json"
	"fmt"
	"gosol/solana_proxy"

	"github.com/slawomir-pryczek/handler_socket2"
)

type Handle_solana_01 struct {
}

func (this *Handle_solana_01) Initialize() {
}

func (this *Handle_solana_01) Info() string {
	return "This plugin will return minimum block numbers for all nodes"
}

func (this *Handle_solana_01) GetActions() []string {
	return []string{"getFirstAvailableBlock", "getBlock"}
}

func (this *Handle_solana_01) HandleAction(action string, data *handler_socket2.HSParams) string {

	if action == "getBlock" {
		block_no := data.GetParamI("block", -1)
		if block_no == -1 {
			return `{"error":"provide block number as &block=123"}`
		}

		client := solana_proxy.GetClientB(false, block_no)
		if client == nil {
			return `{"error":"can't find appropriate client"}`
		}
		ret, is_ok := client.GetBlock(block_no)
		defer client.Release()

		if !is_ok {
			client2 := solana_proxy.GetClient(true)
			if client2 != nil {
				ret, is_ok = client2.GetBlock(block_no)
				client2.Release()
			}
		}
		data.FastReturnBNocopy(ret)
		return ""
	}

	pub, priv := solana_proxy.GetMinBlocks()

	ret := map[string]string{}
	ret["public"] = fmt.Sprintf("%d", pub)
	ret["private"] = fmt.Sprintf("%d", priv)

	_tmp, _ := json.Marshal(ret)
	data.FastReturnBNocopy(_tmp)
	return ""
}
