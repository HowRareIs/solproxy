package handle_solana_01

import (
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
	return []string{"getBlock", "getTransaction", "getBalance", "getTokenSupply"}
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

	if action == "getTransaction" {
		hash := data.GetParam("hash", "")
		if len(hash) == 0 {
			return `{"error":"provide transaction &hash=123"}`
		}

		client := solana_proxy.GetClient(false)
		if client == nil {
			return `{"error":"can't find appropriate client"}`
		}
		ret, is_ok := client.GetTransaction(hash)
		defer client.Release()

		if !is_ok {
			client2 := solana_proxy.GetClient(true)
			if client2 != nil {
				ret, is_ok = client2.GetTransaction(hash)
				client2.Release()
			}
		}
		data.FastReturnBNocopy(ret)
		return ""
	}

	if action == "getBalance" || action == "getTokenSupply" {
		pubkey := data.GetParam("pubkey", "")
		if len(pubkey) == 0 {
			return `{"error":"provide pubkey &pubkey=123, and optionally &commitment="}`
		}
		commitment := data.GetParam("commitment", "")

		client := solana_proxy.GetClient(false)
		if client == nil {
			return `{"error":"can't find appropriate client"}`
		}
		ret, is_ok := client.SimpleCall(action, pubkey, commitment)
		defer client.Release()

		if !is_ok {
			client2 := solana_proxy.GetClient(true)
			if client2 != nil {
				ret, is_ok = client2.SimpleCall(action, pubkey, commitment)
				client2.Release()
			}
		}
		data.FastReturnBNocopy(ret)
		return ""
	}

	return "No function?!"
}
