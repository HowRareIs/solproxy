package solana_proxy

import (
	"gosol/solana_proxy/client"
	"math"
	"sync"
)

var mu sync.RWMutex
var clients []*client.SOLClient

func init() {
	clients = make([]*client.SOLClient, 0, 10)
}

func ClientRegister(c *client.SOLClient) {
	ClientManage(c, math.MaxUint64)
}

func ClientRemove(id uint64) bool {
	return ClientManage(nil, id)
}

func ClientManage(add *client.SOLClient, removeClientID uint64) bool {
	acted := false

	mu.Lock()
	defer mu.Unlock()

	if add != nil && removeClientID == math.MaxUint64 {
		clients = append(clients, add)
		return true
	}

	tmp := make([]*client.SOLClient, 0, len(clients))
	for _, client := range clients {
		if client.GetInfo().ID == removeClientID {
			acted = true
			if add != nil {
				tmp = append(tmp, add)
				add = nil
			}
			continue
		}
		tmp = append(tmp, client)
	}
	if add != nil {
		tmp = append(tmp, add)
	}

	clients = tmp
	return acted
}

func GetMinMaxBlocks() (int, int, int, int) {

	mu.RLock()
	defer mu.RUnlock()

	// a public; b private
	a, b, c, d := -1, -1, -1, -1
	for _, v := range clients {
		info := v.GetInfo()
		if info.Is_disabled {
			continue
		}

		if info.Is_public_node {
			if a == -1 || info.Available_block_first > a {
				a = info.Available_block_first
			}
			if c == -1 || info.Available_block_last < c {
				c = info.Available_block_last
			}
		} else {
			if b == -1 || info.Available_block_first > b {
				b = info.Available_block_first
			}
			if d == -1 || info.Available_block_last < d {
				d = info.Available_block_last
			}
		}
	}
	return a, b, c, d
}
