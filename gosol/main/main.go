package main

import (
	"gosol/passthrough"
	"gosol/solana/handle_solana_01"
	"gosol/solana/handle_solana_info"
	"gosol/solana/handle_solana_raw"
	"gosol/solana_proxy"
	"gosol/solana_proxy/client/throttle"
	"strings"

	"github.com/slawomir-pryczek/handler_socket2"
	"github.com/slawomir-pryczek/handler_socket2/handle_echo"
	"github.com/slawomir-pryczek/handler_socket2/handle_profiler"

	"encoding/json"
	"fmt"
	"os"
)

func _add_node_from_config(node map[string]interface{}) {

	_register := func(endpoint string, public bool, throttle []*throttle.Throttle) {
		if len(endpoint) == 0 {
			return
		}
		max_conn := 50
		if public {
			max_conn = 10
		}
		endpoint = strings.Trim(endpoint, "\r\n\t ")
		solana_proxy.RegisterClient(endpoint, public, max_conn, throttle)
	}

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
		return
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
	_register(url, public, thr)
}

func _read_node_config() {

	fmt.Println("\nReading node config...")
	nodes := (handler_socket2.Config().GetRawData("SOL_NODES", "")).([]interface{})
	if len(nodes) <= 0 {
		fmt.Println("ERROR: No nodes defined, please define at least one solana node to connect to")
		os.Exit(10)
		return
	}

	for _, v := range nodes {
		_add_node_from_config(v.(map[string]interface{}))
	}
	fmt.Println("")
}

func main() {
	_read_node_config()

	num_cpu := runtime.NumCPU() * 2
	runtime.GOMAXPROCS(num_cpu)	// register handlers
	handlers := []handler_socket2.ActionHandler{}
	handlers = append(handlers, &handle_echo.HandleEcho{})
	handlers = append(handlers, &handle_profiler.HandleProfiler{})
	handlers = append(handlers, &handle_solana_raw.Handle_solana_raw{})
	handlers = append(handlers, &handle_solana_01.Handle_solana_01{})
	handlers = append(handlers, &handle_solana_info.Handle_solana_info{})
	handlers = append(handlers, &handle_passthrough.Handle_passthrough{})

	if len(handler_socket2.Config().Get("RUN_SERVICES", "")) > 0 && handler_socket2.Config().Get("RUN_SERVICES", "") != "*" {
		_h_modified := []handler_socket2.ActionHandler{}
		_tmp := strings.Split(handler_socket2.Config().Get("RUN_SERVICES", ""), ",")
		supported := make(map[string]bool)
		for _, v := range _tmp {
			supported[strings.Trim(v, "\r\n \t")] = true
		}

		for _, v := range handlers {
			should_enable := false
			for _, action := range v.GetActions() {
				if supported[action] {
					should_enable = true
					break
				}
			}

			if should_enable {
				_h_modified = append(_h_modified, v)
			}
		}

		handlers = _h_modified
	}

	// start the server
	handler_socket2.RegisterHandler(handlers...)
	handler_socket2.StartServer(strings.Split(handler_socket2.Config().Get("BIND_TO", ""), ","))
}
