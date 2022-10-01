package throttle

import (
	"math"
)

type ThrottleGoup []*Throttle

func (this ThrottleGoup) OnRequest(function_name string) bool {

	for _, throttle := range this {
		if throttle.status_throttled {
			return false
		}
	}

	for _, throttle := range this {
		if !throttle.OnRequest(function_name) {
			return false
		}
	}
	return true
}

func (this ThrottleGoup) OnReceive(data_bytes int) {
	for _, throttle := range this {
		throttle.OnReceive(data_bytes)
	}
}

func (this ThrottleGoup) OnMaintenance(data_bytes int) {
	for _, throttle := range this {
		throttle.OnMaintenance(data_bytes)
	}
}

func (this ThrottleGoup) GetThrottleScore() ThrottleScore {
	ret := ThrottleScore{}
	ret.Score = math.MinInt64
	ret.Throttled = false
	ret.CapacityUsed = 0
	for _, throttle := range this {

		tmp := throttle.GetThrottleScore()
		if tmp.Score > ret.Score {
			ret.Score = tmp.Score
		}
		ret.Throttled = ret.Throttled || tmp.Throttled
		if tmp.CapacityUsed > ret.CapacityUsed {
			ret.CapacityUsed = tmp.CapacityUsed
		}
	}

	return ret
}

func (this ThrottleGoup) GetLimitsLeft() (int, int, int, int) {

	a, b, c, d := math.MaxInt64, math.MaxInt64, math.MaxInt64, 0
	for _, throttle := range this {
		a2, b2, c2, d2 := throttle.GetLimitsLeft()

		if a2 < a {
			a = a2
		}
		if b2 < b {
			b = b2
		}
		if c2 < c {
			c = c2
		}
		if d2 > d {
			d = d2
		}
	}
	return a, b, c, d
}

func (this ThrottleGoup) SetScoreModifier(m int) {
	for _, throttle := range this {
		throttle.score_modifier = m
	}
}
