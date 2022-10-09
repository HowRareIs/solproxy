package handle_solana_01

import (
	"fmt"
	"gosol/solana_proxy"
	"gosol/solana_proxy/client"

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
	cl := sch.GetAnyClient()
	if cl == nil {
		return `{"error":"can't find appropriate client"}`
	}

	if action == "getBlock" {
		block_no := data.GetParamI("block", -1)
		if block_no == -1 {
			return `{"error":"provide block number as &block=123"}`
		}
		sch.SetMinBlock(block_no)
		ret, result := cl.GetBlock(block_no)

		for result != client.R_OK {
			cl = sch.GetPublicClient()
			if cl == nil {
				cl = sch.GetAnyClient()
			}
			if cl == nil {
				return `{"error":"can't find appropriate client (2)"}`
			}
			ret, result = cl.GetBlock(block_no)
		}
		data.FastReturnBNocopy(ret)
		return ""
	}

	if action == "getTransaction" {
		hash := data.GetParam("hash", "")
		if len(hash) == 0 {
			return `{"error":"provide transaction &hash=123"}`
		}

		ret, result := cl.GetTransaction(hash)
		for result != client.R_OK {
			cl = sch.GetPublicClient()
			if cl == nil {
				cl = sch.GetAnyClient()
			}
			if cl == nil {
				return `{"error":"can't find appropriate client (2)"}`
			}

			ret, result = cl.GetTransaction(hash)
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

		ret, result := cl.SimpleCall(action, pubkey, commitment)
		for result != client.R_OK {
			cl = sch.GetPublicClient()
			if cl == nil {
				cl = sch.GetAnyClient()
			}
			if cl == nil {
				return `{"error":"can't find appropriate client (2)"}`
			}
			ret, result = cl.SimpleCall(action, pubkey, commitment)
		}
		data.FastReturnBNocopy(ret)
		return ""
	}

	return "No function?!"
}
