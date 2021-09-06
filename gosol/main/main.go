package main

import (
	"runtime"
	"strings"

	"github.com/slawomir-pryczek/handler_socket2"
	"github.com/slawomir-pryczek/handler_socket2/handle_echo"
	"github.com/slawomir-pryczek/handler_socket2/handle_profiler"

	"gosol/solana/handle_solana_01"
	"gosol/solana_proxy"
)

func main() {

	solana_proxy.RegisterClient(handler_socket2.Config().Get("SOL_NODE_PRIV", ""), 50, false)
	solana_proxy.RegisterClient(handler_socket2.Config().Get("SOL_NODE_PUB", ""), 50, true)

	num_cpu := runtime.NumCPU() * 2
	runtime.GOMAXPROCS(num_cpu)

	// register handlers
	handlers := []handler_socket2.ActionHandler{}
	handlers = append(handlers, &handle_echo.HandleEcho{})
	handlers = append(handlers, &handle_profiler.HandleProfiler{})
	handlers = append(handlers, &handle_solana_01.Handle_solana_01{})

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
