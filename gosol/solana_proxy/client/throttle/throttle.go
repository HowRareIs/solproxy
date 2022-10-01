package throttle

import (
	"time"
)

type LimiterType uint8

const (
	L_REQUESTS        LimiterType = 0
	L_REQUESTS_PER_FN             = 1
	L_DATA_RECEIVED               = 2
)

type Limiter struct {
	t               LimiterType
	maximum         int
	in_time_windows int
}

type Throttle struct {
	limiters []Limiter

	stats_pos                 int
	stats                     []stat
	stats_window_size_seconds int

	score_modifier int

	status_score         int
	status_throttled     bool
	status_capacity_used int
}

func Make() *Throttle {
	return MakeCustom(120, 1)
}

func MakeCustom(window_count, window_size_seconds int) *Throttle {
	ret := &Throttle{}
	ret.limiters = make([]Limiter, 0, 10)

	ret.stats = make([]stat, window_count)
	ret.stats_window_size_seconds = window_size_seconds
	ret.stats_pos = (int(time.Now().Unix()) / ret.stats_window_size_seconds) % len(ret.stats)

	for i := 0; i < len(ret.stats); i++ {
		ret.stats[i].stat_request_by_fn = make(map[string]int)
	}
	return ret
}

func (this *Throttle) AddLimiter(t LimiterType, maximum, time_seconds int) {
	_tw := time_seconds / this.stats_window_size_seconds
	this.limiters = append(this.limiters, Limiter{t, maximum, _tw})
}

func (this *Throttle) SetScoreModifier(m int) {
	this.score_modifier = m
}

/*
func main() {

	fmt.Println("TEST")

	t := Make()
	t = Make()
	t.AddLimiter(L_REQUESTS, 10, 5)
	t.AddLimiter(L_REQUESTS, 20, 3)
	t.AddLimiter(L_REQUESTS_PER_FN, 10, 5)
	t.AddLimiter(L_DATA_RECEIVED, 200000, 10)

	t.OnRequest("TTT")
	t.OnRequest("TTTY")
	t.OnRequest("TTTY")
	t.OnRequest("TTTY")
	t.OnRequest("XXY")
	t.OnRequest("XXTSY")
	t.OnReceive(50000)
	fmt.Println(t.stats)

	fmt.Println("----")
	fmt.Println(t._getThrottleStatus(&t.limiters[0]))
	fmt.Println(t._getThrottleStatus(&t.limiters[1]))
	fmt.Println(t._getThrottleStatus(&t.limiters[2]))

	time.Sleep(1 * time.Second)
	t.OnRequest("TTT")

	fmt.Println("----")
	fmt.Println(t._getThrottleStatus(&t.limiters[0]))
	fmt.Println(t._getThrottleStatus(&t.limiters[1]))
	fmt.Println(t._getThrottleStatus(&t.limiters[2]))

	time.Sleep(1 * time.Second)
	t.OnRequest("TTT")

	fmt.Println("----")
	fmt.Println(t._getThrottleStatus(&t.limiters[0]))
	fmt.Println(t._getThrottleStatus(&t.limiters[1]))
	fmt.Println(t._getThrottleStatus(&t.limiters[2]))

	time.Sleep(1 * time.Second)
	t.OnRequest("TTT")

	fmt.Println("----")
	fmt.Println(t._getThrottleStatus(&t.limiters[0]))
	fmt.Println(t._getThrottleStatus(&t.limiters[1]))
	fmt.Println(t._getThrottleStatus(&t.limiters[2]))
	ts := time.Now().UnixNano()
	x := t._getThrottleScore()
	fmt.Println(time.Now().UnixNano() - ts)

	fmt.Println(x)
}
*/
