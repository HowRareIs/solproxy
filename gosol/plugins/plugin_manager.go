package plugin_manager

import (
	"github.com/slawomir-pryczek/HSServer/handler_socket2"
	"gosol/plugins/common"
	"sync"
	"time"
)

var initialized = false
var plugins = []*common.Plugin{}
var mu = sync.Mutex{}

func register(p *common.Plugin) {
	process := func() {
		mu.Lock()
		_plugins := make([]*common.Plugin, len(plugins))
		copy(_plugins, plugins)
		mu.Unlock()

		for _, p := range _plugins {
			p := p
			go func() { p.Run() }()
		}
	}

	go func() {
		p.Run()
		mu.Lock()
		plugins = append(plugins, p)
		mu.Unlock()
	}()

	mu.Lock()
	if !initialized {
		initialized = true
		go func() {
			for {
				time.Sleep(1 * time.Second)
				process()
			}
		}()
	}
	mu.Unlock()
}

func RegisterAll() {

	/* Genesys plugin
	tmp := genesys.Init("plugin-genesys")
	if tmp != nil {
		register(tmp)
	}*/

	handler_socket2.StatusPluginRegister(func() (string, string) {
		ret := ""
		mu.Lock()
		_plugins := make([]*common.Plugin, len(plugins))
		copy(_plugins, plugins)
		mu.Unlock()

		if len(_plugins) == 0 {
			ret = "No plugins installed!"
		}
		for _, p := range _plugins {
			ret += p.Status() + "\n"
		}

		return "Plugins", "<pre>" + ret + "</pre>"
	})
}
