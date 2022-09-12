package handle_solana_admin

import (
	"encoding/json"
	"fmt"
	"gosol/solana_proxy"
	"gosol/solana_proxy/client"
	"gosol/solana_proxy/client/throttle"
	"math"
	"reflect"
	"strings"
)

func NodeRegister(endpoint string, public bool, probe_time int, throttle []*throttle.Throttle) *client.SOLClient {
	if len(endpoint) == 0 {
		return nil
	}
	max_conn := 50
	if public {
		max_conn = 10
	}
	endpoint = strings.Trim(endpoint, "\r\n\t ")

	if probe_time == -1 {
		if public {
			probe_time = 10
		} else {
			probe_time = 1
		}
	}

	cl := client.MakeClient(endpoint, public, probe_time, max_conn, throttle)
	solana_proxy.ClientManage(cl, math.MaxUint64)
	return cl
}

func _get_cfg_data[T any](node map[string]interface{}, attr string, def T) T {
	if val, ok := node[attr]; ok {
		switch val.(type) {
		case T:
			return val.(T)
		default:
			fmt.Println("Warning: type mismatch for", attr, "attribute is", reflect.TypeOf(val).Name(), ", needs to be ", reflect.TypeOf(new(T)).Name())
		}
	}
	return def
}

func NodeRegisterFromConfig(node map[string]interface{}) *client.SOLClient {

	url := _get_cfg_data(node, "url", "")
	public := _get_cfg_data(node, "public", false)
	score_modifier, _ := _get_cfg_data(node, "score_modifier", json.Number("0")).Int64()
	probe_time, _ := _get_cfg_data(node, "probe_time", json.Number("-1")).Int64()

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
			fmt.Println("Warning: Cannot read throttle settings, skipping throttling")
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
	throttle.ThrottleGoup(thr).SetScoreModifier(int(score_modifier))

	for _, log := range logs {
		fmt.Println(" ", log)
	}

	return NodeRegister(url, public, int(probe_time), thr)
}
