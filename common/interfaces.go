package common

import (
	"math"
)

type ChainInterface interface {
	GetShardID() int
}

const TIMESLOT = 10

func CalculateTimeSlot(time int64) int64 {
	return int64(math.Floor(float64(time / TIMESLOT)))
}
