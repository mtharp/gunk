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

type RelativeConverter struct {
	Rate uint64

	base time.Duration
	rem  uint64
	run  bool
	last uint32
}

func (c *RelativeConverter) Convert(ts uint32) time.Duration {
	// delta from last packet
	if !c.run {
		// set the initial timestamp
		c.last = ts
		c.run = true
	}
	tsDelta := ts - c.last
	c.last = ts
	// delta to duration
	rate := c.Rate
	lo := uint64(tsDelta)*uint64(time.Second) + c.rem
	dur, rem := time.Duration(lo/rate), lo%rate
	c.rem = rem
	t := c.base + dur
	c.base = t
	return t
}
