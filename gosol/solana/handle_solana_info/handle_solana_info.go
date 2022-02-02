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

	/*_round := func(n float64) float64 {
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

		utilization_total := float64(0)
		sum := 0
		for num, v := range sch.GetAll(true, false) {
			key := fmt.Sprintf("throttle-public-%d", num)
			tmp := v.GetThrottle().GetThrottledStatus()
			ret[key] = tmp
			utilization_total += tmp["p_capacity_used"].(float64)
			sum++
		}
		if sum == 0 {
			sum = 1
		}
		ret["percent-capacity-used-public"] = _round(utilization_total / float64(sum))

		utilization_total = 0
		sum = 0
		for num, v := range sch.GetAll(false, false) {
			key := fmt.Sprintf("throttle-private-%d", num)
			tmp := v.GetThrottle().GetThrottledStatus()
			ret[key] = tmp
			utilization_total += tmp["p_capacity_used"].(float64)
			sum++
		}
		if sum == 0 {
			sum = 1
		}
		ret["percent-capacity-used-private"] = _round(utilization_total / float64(sum))

		_tmp, _ := json.Marshal(ret)
		data.FastReturnBNocopy(_tmp)
		return ""
	}*/

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
