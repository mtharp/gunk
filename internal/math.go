package internal

import (
	"math/bits"
	"time"
)

// ToTS converts a duration to a number of clock ticks at a rate of clockRate
func ToTS(t time.Duration, clockRate uint64) uint64 {
	hi, lo := bits.Mul64(uint64(t), clockRate)
	ts, _ := bits.Div64(hi, lo, uint64(time.Second))
	return ts
}

// FromTS converts a tick timestamp to a duration
func FromTS(ts, clockRate uint64) time.Duration {
	hi, lo := bits.Mul64(uint64(ts), uint64(time.Second))
	t, _ := bits.Div64(hi, lo, clockRate)
	return time.Duration(t)
}
