package handler_socket2

import (
	"fmt"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/config"
	"net"
	"os"
	"sync"
)

type ActionHandler interface {
	Initialize()
	Info() string
	GetActions() []string
	HandleAction(action string, data *HSParams) string
}

var action_handlers = make([]ActionHandler, 0)
var actionToHandlerNum = make(map[string]int)

func RegisterHandler(handlers ...ActionHandler) {

	for _, handler := range handlers {
		handler.Initialize()
		action_handlers = append(action_handlers, handler)
	}

	for hindex, handler := range action_handlers {
		for _, action := range handler.GetActions() {
			actionToHandlerNum[action] = hindex
		}
	}
}

type StatusPlugin func() (string, string)

var statusPlugins = make([]StatusPlugin, 0)

func StatusPluginRegister(f StatusPlugin) {
	statusPlugins = append(statusPlugins, f)
}

var boundTo []string = []string{}
var boundMutex sync.Mutex

func StartServer(bind_to []string) {

	compression_ex_read_config()

	var wg sync.WaitGroup
	wg.Add(1)

	for _, bt := range bind_to {
		go func(bt string) {

			switch {
			case bt[0] == 'h':
				startServiceHTTP(bt[1:], handleRequest)

			case bt[0] == 'u':
				startServiceUDP(bt[1:], handleRequest)

			default:
				startService(bt, handleRequest)
			}

			if config.Config().Get("FORCE_START", "") == "1" {
				fmt.Println("WARNING: Can't bind to all interfaces, but FORCE_START in effect")
			} else {
				fmt.Fprintf(os.Stderr, "Cannot bind to: %s or unexpected thread exit\n", bt)
				os.Exit(1)
			}

		}(bt)
	}

	wg.Wait()
}

type handlerFunc func(*HSParams) string

// this is socket handler, it'll just start the socket and then pass flow to serveSocket
// that'll act as socket driver
func startService(bindTo string, handler handlerFunc) {

	tcpAddr, err := net.ResolveTCPAddr("tcp4", bindTo)

	if err != nil {
		fmt.Printf("Error resolving address: %s, %s\n", bindTo, err)
		return
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Printf("Error listening on TCP address: %s, %s\n", bindTo, err)
		return
	}

	fmt.Printf("Socket Service started : %s\n", bindTo)
	boundMutex.Lock()
	boundTo = append(boundTo, "socket:"+bindTo)
	boundMutex.Unlock()

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			continue
		}

		go serveSocket(conn, handler)
	}
}

// limited, self-contained HTTP handler, with limited statistics and no
// compression support
type httpRequest struct {
	req        string
	start_time int64
	end_time   int64
	status     string
}

var httpRequestStatus = make(map[uint64]*httpRequest)
var httpRequestId uint64
var httpStatMutex sync.Mutex

func handleRequest(data *HSParams) string {

	action := data.GetParam("action", "")

	if action == "server-status" {
		return handlerServerStatus(data)
	}

	if config.CfgIsDebug() {
		fmt.Printf("Action %s\n", action)
	}

	if action == "" {
		return "Please specify action (0x1), or ?action=server-status for help"
	}

	if hindex, ok := actionToHandlerNum[action]; ok {
		return (action_handlers[hindex]).HandleAction(action, data)
	}

	return "Please specify action (0x2)"
}
