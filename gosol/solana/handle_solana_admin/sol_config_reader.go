package handle_solana_admin

import (
	"encoding/json"
	"fmt"
	"gosol/solana_proxy"
	"gosol/solana_proxy/client"
	"gosol/solana_proxy/client/throttle"
	"math"
	"strings"
)

func NodeRegister(endpoint string, public bool, throttle []*throttle.Throttle) *client.SOLClient {
	if len(endpoint) == 0 {
		return nil
	}
	max_conn := 50
	if public {
		max_conn = 10
	}
	endpoint = strings.Trim(endpoint, "\r\n\t ")

	cl := client.MakeClient(endpoint, public, max_conn, throttle)
	solana_proxy.ClientManage(cl, math.MaxUint64)
	return cl
}

func NodeRegisterFromConfig(node map[string]interface{}) *client.SOLClient {

	public := false
	url := ""
	score_modifier := 0
	if val, ok := node["url"]; ok {
		switch val.(type) {
		case string:
			url = val.(string)
		}
	}

	if val, ok := node["public"]; ok {
		switch val.(type) {
		case bool:
			public = val.(bool)
		default:
			fmt.Println("Warning: type mismatch for public attribute, needs to be true/false")
		}
	}

	if val, ok := node["score_modifier"]; ok {
		switch val.(type) {
		case json.Number:
			tmp, _ := val.(json.Number).Int64()
			score_modifier = int(tmp)
		default:
			fmt.Println("Warning: type mismatch for score_adjust attribute, needs to be number")
		}
	}

	if url == "" {
		fmt.Println("Cannot read node config (no url) ... skipping")
		return nil
	}

	thr := ([]*throttle.Throttle)(nil)
	logs := []string{}
	fmt.Printf("## Node: %s Public: %v, score modifier: %d\n", url, public, score_modifier)

	if val, ok := node["throttle"]; ok {
		switch val.(type) {
		case string:
			thr, logs = throttle.MakeFromConfig(val.(string))
		default:
			fmt.Println("Warning: Cannot read throttle settings, skipping")
		}
	} else {
		if public {
			thr, logs = throttle.MakeForPublic()
		}
	}

	if thr == nil {
		thr = make([]*throttle.Throttle, 0, 1)
		thr = append(thr, throttle.Make())
		logs = append(logs, "Throttling disabled")
	}
	throttle.ThrottleGoup(thr).SetScoreModifier(score_modifier)

	for _, log := range logs {
		fmt.Println(" ", log)
	}

	return NodeRegister(url, public, thr)
}
