package irtmp

import (
	"time"

	"github.com/nareix/joy4/av"
)

const (
	rateMatch    = time.Millisecond
	residueMatch = time.Microsecond
)

var stdRates = []time.Duration{
	time.Second / 30,
	time.Second / 60,

	time.Second / 10,
	time.Second / 15,
	time.Second / 20,
	time.Second / 24,
	time.Second / 25,
	time.Second / 48,
	time.Second / 50,
	time.Second / 120,
	time.Second / 144,
}

// DeJitter fixes timestamps that got rounded to the nearest millisecond.
// It assumes a standard framerate is in use and nudges the timestamp on each packet.
// This doesn't work for 29.97/59.94 fps but it doesn't make it notably worse either.
type DeJitter struct {
	lastV time.Duration
	lastA time.Duration
}

func applyRate(pkt *av.Packet, gap, candidate time.Duration) bool {
	delta := candidate - gap
	if delta < -rateMatch || delta > rateMatch {
		return false
	}
	t := pkt.Time + delta
	// integer math means rounding errors, so round up to nearest second as appropriate
	if remainder := t % time.Second; remainder > time.Second-residueMatch {
		t += time.Second - remainder
	}
	pkt.Time = t
	return true
}

func (j *DeJitter) ModifyPacket(pkt *av.Packet, streams []av.CodecData, videoidx int, audioidx int) (drop bool, err error) {
	switch int(pkt.Idx) {
	case videoidx:
		gap := pkt.Time - j.lastV
		if gap == 0 {
			return
		}
		for _, candRate := range stdRates {
			if applyRate(pkt, gap, candRate) {
				break
			}
		}
		j.lastV = pkt.Time
	case audioidx:
		gap := pkt.Time - j.lastA
		if gap == 0 {
			return
		}
		acd := streams[pkt.Idx].(av.AudioCodecData)
		duration, _ := acd.PacketDuration(pkt.Data)
		if duration != 0 {
			applyRate(pkt, gap, duration)
		}
		j.lastA = pkt.Time
	}
	return
}
