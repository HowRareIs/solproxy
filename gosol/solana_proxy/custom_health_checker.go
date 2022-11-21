package solana_proxy

import (
	"fmt"
	"github.com/slawomir-pryczek/HSServer/handler_socket2"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/config"
	"gosol/solana_proxy/client"
	"strings"
	"sync"
	"time"
)

type custom_health_checker struct {
	mu sync.Mutex

	run_every       int64
	max_data_age_ms int64
	max_block_lag   int

	_log string
}

var cc custom_health_checker

func init() {

	cfg := config.Config()
	has_custom_checker, err := cfg.ValidateAttribs("CUSTOM_HEALTH_CHECKER", []string{"run_every", "max_block_lag", "max_data_age_ms"})
	if err != nil {
		panic("Custom chealth checker config error. " + err.Error())
	}
	if !has_custom_checker {
		return
	}

	run_every, err := config.Config().GetSubattrInt("CUSTOM_HEALTH_CHECKER", "run_every")
	if err != nil {
		panic(err)
	}
	max_block_lag, err := config.Config().GetSubattrInt("CUSTOM_HEALTH_CHECKER", "max_block_lag")
	if err != nil {
		panic(err)
	}
	max_data_age_ms, err := config.Config().GetSubattrInt("CUSTOM_HEALTH_CHECKER", "max_data_age_ms")
	if err != nil {
		panic(err)
	}

	cc = custom_health_checker{}
	cc.run_every = int64(run_every)
	cc.max_block_lag = max_block_lag
	cc.max_data_age_ms = int64(max_data_age_ms)

	go func() {
		last := time.Now().Unix()
		for {
			if t := time.Now().Unix(); t-last < cc.run_every {
				continue
			} else {
				last = t
			}
			_run_custom_check()
		}
	}()

	handler_socket2.StatusPluginRegister(func() (string, string) {
		ret := "Custom health plugin will pause nodes when they start lagging\n"
		ret += fmt.Sprintf("run_every: %d - run the check every X seconds\n", cc.run_every)
		ret += fmt.Sprintf("max_block_lag: %d - maximum number of blocks which a node can lag behind, before being paused\n", cc.max_block_lag)
		ret += fmt.Sprintf("max_data_age_ms: %d - maximum age of highest block data (in milliseconds), if max block data is older, it'll be re-fetched\n", cc.max_data_age_ms)

		ret += "--------\n"
		ret += cc._log
		if len(cc._log) == 0 {
			ret += "Waiting for data"
		}

		return "Solana Proxy - Custom Health Plugin", "<pre>" + ret + "</pre>"
	})
}

func _run_custom_check() {

	max_age_ms := cc.max_data_age_ms
	max_block_lag := cc.max_block_lag

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

			_comm := "Never updated"
			if _age_ms < 24*3600*1000 {
				_comm = fmt.Sprintf("%.2fs", float64(_age_ms)/1000.0)
			}
			status[num] = fmt.Sprintf("(Trying refresh - %s)", _comm)
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

	log := ""
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

		log += fmt.Sprintf("Node #%d %s Score: %d, Highest Block: %d/%d Max (%d diff) (%.2fs Age) %s\n",
			num, _is_ok, info.Score,
			info.Available_block_last, max_block, _diff,
			_age_ms, status[num])
	}

	cc.mu.Lock()
	cc._log = log
	cc.mu.Unlock()
}
