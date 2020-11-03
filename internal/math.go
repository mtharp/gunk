package internal

import (
	"math/bits"
	"time"
)

// ToTS converts a duration to a number of clock ticks at a rate of clockRate
func ToTS(t time.Duration, clockRate uint64) uint64 {
	hi, lo := bits.Mul64(uint64(t), clockRate)
	ts, rem := bits.Div64(hi, lo, uint64(time.Second))
	if rem >= uint64(time.Second/2) {
		// round up
		ts++
	}
	return ts
}

// FromTS converts a tick timestamp to a duration
func FromTS(ts, clockRate uint64) time.Duration {
	hi, lo := bits.Mul64(uint64(ts), uint64(time.Second))
	t, rem := bits.Div64(hi, lo, clockRate)
	if rem > clockRate/2 {
		// round up
		t++
	}
	return time.Duration(t)
}
