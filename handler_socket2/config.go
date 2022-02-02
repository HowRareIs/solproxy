package handler_socket2

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var _config atomic.Value

var cfg_initialized = false
var cfg_mu sync.Mutex

func init() {
	ReadConfig()

	go func() {
		for {
			time.Sleep(5 * time.Second)

			_cfg := Config()
			st, err := os.Stat(_cfg.cfg_file_path)
			if err != nil {
				continue
			}

			cfg_mu.Lock()
			_ci := cfg_initialized
			cfg_mu.Unlock()
			if !_ci {
				continue
			}

			file_changed := false
			file_changed = file_changed || _cfg.cfg_file_size != st.Size()
			file_changed = file_changed || _cfg.cfg_file_modified != st.ModTime().Unix()
			if !file_changed {
				continue
			}

			fmt.Println("\n\nConfig file changed, re-reading configuration!")
			tmp, err := _cfg_load_config()
			if err != nil {
				continue
			}
			tmp._cfg_local_interfaces()
			tmp._cfg_conditional_config()
			_config.Store(tmp)
		}
	}()
}

func ReadConfig() {

	cfg_mu.Lock()
	defer cfg_mu.Unlock()
	if cfg_initialized {
		return
	}
	cfg_initialized = true

	tmp, err := _cfg_load_config()
	if err != nil {
		fmt.Println("FATAL Error opening configuration file conf.json:", err)
		os.Exit(1)
	}

	tmp._cfg_local_interfaces()
	tmp._cfg_conditional_config()
	_config.Store(tmp)
}

func Config() *cfg {
	ret := _config.Load()
	if ret != nil {
		return ret.(*cfg)
	}

	ReadConfig()
	return _config.Load().(*cfg)
}

func (this *cfg) GetIPDistance(remote_addr string) byte {

	is_local := false
	for _, ip := range this.local_interfaces {
		if strings.Compare(ip, remote_addr) == 0 {
			is_local = true
			break
		}
	}

	if is_local {
		return 0
	}
	return 1
}

func (this *cfg) GetRawData(attr string, def string) interface{} {
	if val, ok := this.raw_data[attr]; ok {
		return val
	}
	return def
}

func (this *cfg) Get(attr, def string) string {
	if val, ok := this.config[attr]; ok {
		return val
	}
	return def
}

func (this *cfg) GetB(attr string) bool {

	if val, ok := this.config[attr]; ok && val == "1" {
		return true
	}
	return false
}

func (this *cfg) GetI(attr string, def int) int {

	if _, ok := this.config[attr]; !ok {
		return def
	}

	if ret, err := strconv.ParseInt(this.config[attr], 10, 64); err == nil {
		return int(ret)
	}
	return def
}

func (this *cfg) GetCompressionThreshold() int {
	return this.compression_threshold
}
