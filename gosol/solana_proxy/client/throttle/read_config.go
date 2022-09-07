package throttle

import (
	"fmt"
	"strconv"
	"strings"
)

func MakeFromConfig(config string) ([]*Throttle, []string) {
	logs := make([]string, 0)
	error := func(a ...interface{}) ([]*Throttle, []string) {
		logs = append(logs, fmt.Sprint(a...))
		return nil, logs
	}
	log := func(a ...interface{}) {
		logs = append(logs, fmt.Sprint(a...))
	}

	type tw struct {
		time_window_size  int
		time_window_count int
	}
	_get_timewindow_len := func(stat_time int) tw {
		if stat_time <= 120 {
			return tw{1, 120}
		}
		if stat_time <= 1200 {
			if stat_time%10 == 0 {
				return tw{10, 120}
			}

			log("Warning: throttling time >120s and not divisible by 10, may be suboptimal. Consider changing it")
		}

		start := stat_time / 100
		end := stat_time / 200
		for i := start; i < end; i++ {
			if stat_time%i == 0 {
				return tw{i, stat_time / i}
			}
		}
		log("Warning: throttling time can't be sliced into windows, consider using ",
			((stat_time/120)+1)*120, " instead of ", stat_time)
		return tw{stat_time / 120, 125}
	}
	tws := make(map[tw]*Throttle, 0)
	_get_thr := func(throttle_time int) *Throttle {
		_k := _get_timewindow_len(throttle_time)
		if ret, ok := tws[_k]; ok {
			return ret
		}
		tws[_k] = MakeCustom(_k.time_window_count, _k.time_window_size)
		return tws[_k]
	}

	for _, v := range strings.Split(config, ";") {
		log("Processing throttle config: ", v)
		v := strings.Split(v, ",")
		if len(v) < 3 {
			return error("Error configuring throttling:", v, "...  needs to have 3 parameters: type,limit,time_seconds")
		}
		for kk, vv := range v {
			v[kk] = strings.Trim(vv, "\r\n\t ")
		}

		if v[0] != "r" && v[0] != "f" && v[0] != "d" {
			return error("Error configuring throttling:", v, "... type needs to be [r]equests, [f]unctions, [d]ata received")
		}

		t_limit, _ := strconv.Atoi(v[1])
		t_time, _ := strconv.Atoi(v[2])
		thr := _get_thr(t_time)
		if t_limit <= 0 {
			return error("Error configuring throttling:", t_limit, "... limit needs to be > 0")
		}
		if t_time <= 0 {
			return error("Error configuring throttling:", t_time, "... time needs to be > 0")
		}

		if v[0] == "r" {
			thr.AddLimiter(L_REQUESTS, t_limit, t_time)
			log("Throttling requests ", t_limit, "/", t_time, "seconds")
		}
		if v[0] == "f" {
			thr.AddLimiter(L_REQUESTS_PER_FN, t_limit, t_time)
			log("Throttling requests for single function ", t_limit, "/", t_time, "seconds")
		}
		if v[0] == "d" {
			thr.AddLimiter(L_DATA_RECEIVED, t_limit, t_time)
			log("Throttling data received ", t_limit, "bytes/", t_time, "seconds")
		}
	}

	ret := make([]*Throttle, 0, len(tws))
	for _, v := range tws {
		ret = append(ret, v)
	}
	return ret, logs
}

func MakeForPublic() ([]*Throttle, []string) {
	thr := Make()
	thr.AddLimiter(L_REQUESTS, 70, 10)
	thr.AddLimiter(L_REQUESTS_PER_FN, 20, 10)
	thr.AddLimiter(L_DATA_RECEIVED, 75*1000*1000, 30)

	return []*Throttle{thr}, []string{"Adding standard throttle for public nodes"}
}
