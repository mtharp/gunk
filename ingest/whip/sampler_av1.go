package whip

import (
	"github.com/pion/rtp/codecs"
)

func newAV1Parser() *av1Parser {
	return new(av1Parser)
}

type av1Parser struct {
	codecs.AV1Packet
}

func (av1Parser) IsPartitionHead(_ []byte) bool {
	return true // TODO?
}

func (av1Parser) IsPartitionTail(marker bool, _ []byte) bool {
	return marker
}

func (p *av1Parser) OnPacketHead(pkt any) any {
	av1Pkt := pkt.(*av1Parser)
	return packetMeta{IsKeyFrame: av1Pkt.N}
}
