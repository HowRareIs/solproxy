package handle_solana_info

import (
	"encoding/json"
	"fmt"
	"gosol/solana_proxy"

	"github.com/slawomir-pryczek/handler_socket2"
)

type Handle_solana_info struct {
}

func (this *Handle_solana_info) Initialize() {
}

func (this *Handle_solana_info) Info() string {
	return "This plugin will return solana nodes information"
}

func (this *Handle_solana_info) GetActions() []string {
	return []string{"getFirstAvailableBlock", "getSolanaInfo"}
}

func (this *Handle_solana_info) HandleAction(action string, data *handler_socket2.HSParams) string {

	if action == "getSolanaInfo" {

		pub, priv := solana_proxy.GetMinBlocks()

		ret := map[string]interface{}{}
		ret["first_available_block"] = map[string]string{
			"public":  fmt.Sprintf("%d", pub),
			"private": fmt.Sprintf("%d", priv)}

		ret["throttle-public"] = solana_proxy.GetClient(true).GetThrottledStatus()
		ret["throttle-private"] = solana_proxy.GetClient(false).GetThrottledStatus()

		_tmp, _ := json.Marshal(ret)
		data.FastReturnBNocopy(_tmp)
		return ""
	}

	if action == "getFirstAvailableBlock" {

		pub, priv := solana_proxy.GetMinBlocks()

		ret := map[string]string{}
		ret["public"] = fmt.Sprintf("%d", pub)
		ret["private"] = fmt.Sprintf("%d", priv)

		_tmp, _ := json.Marshal(ret)
		data.FastReturnBNocopy(_tmp)
		return ""
	}

	return "No function ?!"
}
