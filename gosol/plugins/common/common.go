package common

import (
	"sync"
	"time"
)

type pluginInterface interface {
	Run(age_ms int) bool
	Status() string
}

func (this *Plugin) Status() string {
	return this.p.Status()
}

func (this *Plugin) Run() bool {
	now := time.Now().UnixMilli()

	// run only once, plus check age
	this.mu.Lock()
	if this.is_running {
		this.counter_skip++
		this.mu.Unlock()
		return false
	}
	this.is_running = true
	age := int(now - this.last_run_time)
	if this.last_run_time == 0 {
		age = -1
	}
	this.mu.Unlock()

	// run plugin processing
	_r := this.p.Run(age)
	this.mu.Lock()
	if _r {
		this.last_run_time = now
	}
	this.is_running = false
	this.counter_run++
	this.mu.Unlock()
	return true
}

type Plugin struct {
	p  pluginInterface
	mu sync.Mutex

	is_running   bool
	counter_run  int
	counter_skip int

	last_run_time int64
}

func PluginFactory(p pluginInterface) *Plugin {
	if p == nil {
		return nil
	}
	return &Plugin{p: p}
}
