package solana_proxy

import (
	"gosol/solana_proxy/client"
	"gosol/solana_proxy/client/throttle"
	"sync"
)

var mu sync.RWMutex
var clients []*client.SOLClient

func init() {
	clients = make([]*client.SOLClient, 0, 10)
}

func RegisterClient(endpoint string, is_public_node bool, max_conns int, throttle *throttle.Throttle) {
	cl := client.MakeClient(endpoint, is_public_node, max_conns, throttle)

	mu.Lock()
	clients = append(clients, cl)
	mu.Unlock()
}

func GetMinBlocks() (int, int) {

	mu.RLock()
	defer mu.RUnlock()

	// a public; b private
	a, b := -1, -1
	for _, v := range clients {
		info := v.GetInfo()
		if info.Is_disabled {
			continue
		}

		if info.Is_public_node {
			if a == -1 || info.First_available_block < a {
				a = info.First_available_block
			}
		} else {
			if b == -1 || info.First_available_block < b {
				b = info.First_available_block
			}
		}
	}
	return a, b
}
