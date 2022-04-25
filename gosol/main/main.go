package main

import (
	"runtime"
	"strings"

	"gosol/passthrough"
	"gosol/solana/handle_solana_01"
	"gosol/solana/handle_solana_info"
	"gosol/solana/handle_solana_raw"
	"gosol/solana_proxy"
	"gosol/solana_proxy/client/throttle"

	"github.com/slawomir-pryczek/handler_socket2"
	"github.com/slawomir-pryczek/handler_socket2/handle_echo"
	"github.com/slawomir-pryczek/handler_socket2/handle_profiler"

	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

func _read_node_config() {
	_register := func(endpoint string, public bool, throttle *throttle.Throttle) {
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

	fmt.Println("\nReading node config...")
	nodes := (handler_socket2.Config().GetRawData("SOL_NODES", "")).([]interface{})
	if len(nodes) <= 0 {
		fmt.Println("ERROR: No nodes defined, please define at least one solana node to connect to")
		os.Exit(10)
		return
	}

	num := 0
	for _, v := range nodes {
		tmp := v.(map[string]interface{})

		public := false
		url := ""
		score_modifier := 0
		if val, ok := tmp["url"]; ok {
			switch val.(type) {
			case string:
				url = val.(string)
			}
		}

		if val, ok := tmp["public"]; ok {
			switch val.(type) {
			case bool:
				public = val.(bool)
			default:
				fmt.Println("Warning: type mismatch for public attribute, needs to be true/false")
			}
		}

		if val, ok := tmp["score_modifier"]; ok {
			switch val.(type) {
			case json.Number:
				tmp, _ := val.(json.Number).Int64()
				score_modifier = int(tmp)
			default:
				fmt.Println("Warning: type mismatch for score_adjust attribute, needs to be number")
			}
		}

		if url == "" {
			fmt.Println("Cannot read node config:", tmp, "... skipping")
			continue
		}

		thr := throttle.Make()
		fmt.Printf("#%d: %s Public: %v, score modifier: %d\n", num, url, public, score_modifier)
		if val, ok := tmp["throttle"]; ok {
			tmp := ""
			switch val.(type) {
			case string:
				tmp, _ = val.(string)
			}

			for _, v := range strings.Split(tmp, ";") {
				v := strings.Split(v, ",")
				if len(v) < 3 {
					fmt.Println(" Error configuring throttling:", v, "...  needs to have 3 parameters: type,limit,time_seconds")
					continue
				}
				for kk, vv := range v {
					v[kk] = strings.Trim(vv, "\r\n\t ")
				}

				if v[0] != "r" && v[0] != "f" && v[0] != "d" {
					fmt.Println(" Error configuring throttling:", v, "... type needs to be [r]equests, [f]unctions, [d]ata received")
					continue
				}

				t_limit, _ := strconv.Atoi(v[1])
				t_time, _ := strconv.Atoi(v[2])
				if t_limit <= 0 {
					fmt.Println(" Error configuring throttling:", t_limit, "... limit needs to be >= 0")
					continue
				}
				if t_time <= 0 {
					fmt.Println(" Error configuring throttling:", t_time, "... time needs to be >= 0")
					continue
				}

				if v[0] == "r" {
					thr.AddLimiter(throttle.L_REQUESTS, t_limit, t_time)
					fmt.Println(" Throttling requests", t_limit, "/", t_time, "seconds")
				}
				if v[0] == "f" {
					thr.AddLimiter(throttle.L_REQUESTS_PER_FN, t_limit, t_time)
					fmt.Println(" Throttling requests for single function", t_limit, "/", t_time, "seconds")
				}
				if v[0] == "d" {
					thr.AddLimiter(throttle.L_DATA_RECEIVED, t_limit, t_time)
					fmt.Println(" Throttling data received", t_limit, "bytes /", t_time, "seconds")
				}
			}
		} else {
			if public {
				thr.AddLimiter(throttle.L_REQUESTS, 90, 12)
				thr.AddLimiter(throttle.L_REQUESTS_PER_FN, 33, 12)
				thr.AddLimiter(throttle.L_DATA_RECEIVED, 95*1000*1000, 32)
				fmt.Println(" Adding standard throttle for public nodes")
			}
		}
		fmt.Println("")

		thr.SetScoreModifier(score_modifier)
		_register(url, public, thr)
		num++
	}
}

func main() {
	_read_node_config()

	num_cpu := runtime.NumCPU() * 2
	runtime.GOMAXPROCS(num_cpu)

	// register handlers
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
			for _, action := range handler_socket2.ActionHandler(v).GetActions() {
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
