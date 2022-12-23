package handle_solana_01

import (
	"gosol/solana_proxy"
	"gosol/solana_proxy/client"
	"strings"

	"github.com/slawomir-pryczek/HSServer/handler_socket2"
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

	mstv_i := 0
	if action == "getBlock" || action == "getTransaction" {
		mstv := data.GetParam("maxSupportedTransactionVersion", "")
		if len(mstv) > 0 {
			mstv_i = -2
			if strings.EqualFold("legacy", mstv) {
				mstv_i = -1
			}
			if mstv == "0" {
				mstv_i = 0
			}
		}
		if mstv_i == -2 {
			return `{"error":"maxSupportedTransactionVersion needs to be 0 or legacy"}`
		}
	}

	if action == "getBlock" {
		block_no := data.GetParamI("block", -1)
		if block_no == -1 {
			return `{"error":"provide block number as &block=123"}`
		}

		sch.SetMinBlock(block_no)
		ret, result := cl.GetBlock(block_no, mstv_i)

		for result != client.R_OK {
			cl = sch.GetPublicClient()
			if cl == nil {
				cl = sch.GetAnyClient()
			}
			if cl == nil {
				return `{"error":"can't find appropriate client (2)"}`
			}
			ret, result = cl.GetBlock(block_no, mstv_i)
		}
		data.FastReturnBNocopy(ret)
		return ""
	}

	if action == "getTransaction" {
		hash := data.GetParam("hash", "")
		if len(hash) == 0 {
			return `{"error":"provide transaction &hash=123"}`
		}

		ret, result := cl.GetTransaction(hash, mstv_i)
		for result != client.R_OK {
			cl = sch.GetPublicClient()
			if cl == nil {
				cl = sch.GetAnyClient()
			}
			if cl == nil {
				return `{"error":"can't find appropriate client (2)"}`
			}

			ret, result = cl.GetTransaction(hash, mstv_i)
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
