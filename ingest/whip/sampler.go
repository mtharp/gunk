package whip

import (
	"github.com/nareix/joy4/av"
	"github.com/pion/rtp"
)

type sampleParser interface {
	rtp.Depacketizer
	// SampleBuilder will invoke this on the first packet of each frame, which
	// should determine whether it is a keyframe and return a packetMeta
	OnPacketHead(any) any
}

type packetMeta struct {
	IsKeyFrame bool
}

// Copy flags into av.Packet
func (m packetMeta) Update(pkt *av.Packet) {
	pkt.IsKeyFrame = m.IsKeyFrame
}

// no-op implementation for audio codecs
type audioParser struct {
	rtp.Depacketizer
}

func (audioParser) OnPacketHead(any) any {
	return packetMeta{}
}
