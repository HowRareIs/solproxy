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

	sch := solana_proxy.MakeScheduler()
	if data.GetParamI("public", 0) == 1 {
		sch.ForcePublic(true)
	}
	if data.GetParamI("private", 0) == 1 {
		sch.ForcePrivate(true)
	}
	client := sch.GetAnyClient()
	if client == nil {
		return `{"error":"can't find appropriate client"}`
	}

	if action == "getBlock" {
		block_no := data.GetParamI("block", -1)
		if block_no == -1 {
			return `{"error":"provide block number as &block=123"}`
		}

		sch.SetMinBlock(block_no)
		ret, is_ok := client.GetBlock(block_no)
		if !is_ok {
			client := sch.GetPublicClient()
			if client != nil {
				ret, _ = client.GetBlock(block_no)
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

		ret, is_ok := client.GetTransaction(hash)
		if !is_ok {
			client = sch.GetPublicClient()
			if client != nil {
				ret, _ = client.GetTransaction(hash)
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

		ret, is_ok := client.SimpleCall(action, pubkey, commitment)
		if !is_ok {
			client = sch.GetPublicClient()
			if client != nil {
				ret, _ = client.SimpleCall(action, pubkey, commitment)
			}
		}
		data.FastReturnBNocopy(ret)
		return ""
	}

	return "No function?!"
}
