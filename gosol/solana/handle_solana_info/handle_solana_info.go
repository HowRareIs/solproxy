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

	_round := func(n float64) float64 {
		tmp := int(n * 1000.0)
		return float64(tmp/100) / 10.0
	}

	if action == "getSolanaInfo" {
		pub, priv := solana_proxy.GetMinBlocks()

		ret := map[string]interface{}{}
		ret["first_available_block"] = map[string]string{
			"public":  fmt.Sprintf("%d", pub),
			"private": fmt.Sprintf("%d", priv)}

		sch := solana_proxy.MakeScheduler()
		if data.GetParamI("public", 0) == 1 {
			sch.ForcePublic(true)
		}
		if data.GetParamI("private", 0) == 1 {
			sch.ForcePrivate(true)
		}

		// calculate limits
		var limits_pub_left = [5]int{0, 0, 0, 0, 0}
		var limits_priv_left = [5]int{0, 0, 0, 0, 0}
		tmp := [4]int{}
		for num, v := range sch.GetAll(true, true) {
			tmp[0], tmp[1], tmp[2], tmp[3] = v.GetThrottleLimitsLeft()
			for i := 0; i < 4; i++ {
				limits_pub_left[i] += tmp[i]
			}
			limits_pub_left[4]++
			ret[fmt.Sprintf("pub-%d", num)] = tmp
		}
		for num, v := range sch.GetAll(false, true) {
			tmp[0], tmp[1], tmp[2], tmp[3] = v.GetThrottleLimitsLeft()
			for i := 0; i < 4; i++ {
				limits_priv_left[i] += tmp[i]
			}
			limits_priv_left[4]++
			ret[fmt.Sprintf("priv-%d", num)] = tmp
		}

		// generate JSON
		_gen := func(data [5]int) map[string]interface{} {
			ret := map[string]interface{}{}
			ret["requests_left"] = data[0]
			ret["requests_single_left"] = data[1]
			ret["byte_received_left"] = data[2]
			if data[4] == 0 {
				ret["utilization_percent"] = 0
			} else {
				ret["utilization_percent"] = _round((float64(data[3]) / float64(data[4])) / 100.0)
			}
			ret["node_count"] = data[4]
			return ret
		}
		ret["public"] = _gen(limits_pub_left)
		ret["private"] = _gen(limits_priv_left)
		ret["comment"] = "Per node data is requests left / requests single left / bytes reveived left / utilization percentage"
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
