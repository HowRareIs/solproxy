package solana_proxy

import (
	"fmt"
	"gosol/solana_proxy/client"
	"strings"
	"sync"
	"time"
)

func init() {
	go func() {
		run_every := int64(5)
		last := time.Now().Unix()
		for {
			if t := time.Now().Unix(); t-last < run_every {
				continue
			} else {
				last = t
			}
			_run_custom_check()
		}
	}()
}

func _run_custom_check() {

	max_age_ms := int64(30000)
	max_block_lag := 1000

	mu.RLock()
	infos := make([]*client.Solclientinfo, 0, len(clients))
	for _, client := range clients {
		infos = append(infos, client.GetInfo())
	}
	mu.RUnlock()

	status := make(map[int]string, 0)
	max_block := 0
	wg := sync.WaitGroup{}
	info_mutex := sync.Mutex{}
	for num, info := range infos {
		_age_ms := time.Now().UnixMilli() - info.Available_block_last_ts
		if _age_ms < max_age_ms {
			continue
		}

		// Check if we still can run this client
		_client := (*client.SOLClient)(nil)
		mu.RLock()
		if num < len(clients) {
			wg.Add(1)
			_client = clients[num]
			status[num] = "(Should refresh)"
		}
		mu.RUnlock()
		if _client == nil {
			continue
		}

		// Get max blocks for all clients which need it
		num := num
		go func() {
			_client.GetLastAvailableBlock()
			_info := _client.GetInfo()
			info_mutex.Lock()
			infos[num] = _info
			info_mutex.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()

	//Get maximum block
	for _, info := range infos {
		if info.Available_block_last > max_block {
			max_block = info.Available_block_last
		}
	}

	mu.RLock()
	defer mu.RUnlock()

	for num, info := range infos {
		is_ok := max_block-info.Available_block_last <= max_block_lag
		_is_ok := "OK     "
		if !is_ok {
			_is_ok = "LAGGING"
		}

		for _, client := range clients {
			if strings.Compare(client.GetEndpoint(), info.Endpoint) == 0 {
				if is_ok {
					client.SetPaused(!is_ok, "")
					continue
				}

				if info.Available_block_last_ts == 0 {
					client.SetPaused(!is_ok, "Paused by Custom Health Checker.\nCan't get last block")
					continue
				}
				client.SetPaused(!is_ok, fmt.Sprintf("Paused by Custom Health Checker.\nNode is lagging behind %d blocks (%d max)",
					max_block-info.Available_block_last, max_block_lag))
			}
		}

		_age_ms := float64((time.Now().UnixMilli() - info.Available_block_last_ts)) / 1000
		_diff := max_block - info.Available_block_last
		fmt.Printf("Node #%d %s Score: %d, Highest Block: %d/%d Max (%d diff) (%.2fs Age) %s\n",
			num, _is_ok, info.Score,
			info.Available_block_last, max_block, _diff,
			_age_ms, status[num])
	}

}
