package whip

import (
	"fmt"

	"eaglesong.dev/gunk/h264util"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/pion/rtp/codecs"
)

func newH264Parser(codecData chan<- av.CodecData) *h264Parser {
	return &h264Parser{
		H264Packet: codecs.H264Packet{
			// convert to AVCC
			IsAVC: true,
		},
		cd: codecData,
	}
}

type h264Parser struct {
	codecs.H264Packet

	isKeyFrame bool
	sps, pps   []byte
	cd         chan<- av.CodecData
}

func (p *h264Parser) Unmarshal(payload []byte) ([]byte, error) {
	// disassemble/reassemble fragments
	assembled, err := p.H264Packet.Unmarshal(payload)
	if err != nil {
		return nil, err
	}
	// stash the bits we need
	p.isKeyFrame = false
	nalus, _ := h264parser.SplitNALUs(assembled)
	for _, nalu := range nalus {
		switch h264util.NALType(nalu) {
		case h264util.TypeIDR:
			p.isKeyFrame = true
		case h264util.TypeSequenceParameter:
			if p.cd != nil {
				p.sps = nalu
			}
		case h264util.TypePictureParameter:
			if p.cd != nil {
				p.pps = nalu
			}
		}
	}
	// finish the CodecData
	if p.cd != nil && len(p.pps) > 0 && len(p.sps) > 0 {
		cd, err := h264parser.NewCodecDataFromSPSAndPPS(p.sps, p.pps)
		if err != nil {
			return nil, fmt.Errorf("parsing h264 codec data: %w", err)
		}
		p.cd <- cd
		close(p.cd)
		p.cd = nil
	}
	return assembled, nil
}

func (*h264Parser) OnPacketHead(pkt any) any {
	h264pkt := pkt.(*h264Parser)
	return packetMeta{IsKeyFrame: h264pkt.isKeyFrame}
}
