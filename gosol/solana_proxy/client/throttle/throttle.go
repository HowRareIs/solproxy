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
	t            LimiterType
	maximum      int
	time_seconds int
}

type Throttle struct {
	limiters []Limiter

	stats_pos int
	stats     []stat

	score_modifier int

	status_score         int
	status_disabled      bool
	status_capacity_used int
}

func Make() *Throttle {

	ret := &Throttle{}
	ret.limiters = make([]Limiter, 0, 10)

	ret.stats = make([]stat, 60)
	ret.stats_pos = int(time.Now().Unix()) % len(ret.stats)

	for i := 0; i < len(ret.stats); i++ {
		ret.stats[i].stat_request_by_fn = make(map[string]int)
	}
	return ret
}

func (this *Throttle) AddLimiter(t LimiterType, maximum, time_seconds int) {
	this.limiters = append(this.limiters, Limiter{t, maximum, time_seconds})
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
