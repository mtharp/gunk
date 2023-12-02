package whip

import (
	"fmt"

	"eaglesong.dev/gunk/h264util"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/h264parser"
)

type sampler interface {
	Update(*av.Packet) error
}

type h264sampler struct {
	sps, pps []byte

	cd chan<- av.CodecData
}

func (s *h264sampler) Update(pkt *av.Packet) error {
	nalus, _ := h264parser.SplitNALUs(pkt.Data)
	for _, nalu := range nalus {
		switch h264util.NALType(nalu) {
		case h264util.TypeIDR:
			pkt.IsKeyFrame = true
		case h264util.TypeSequenceParameter:
			if s.cd != nil {
				s.sps = nalu
			}
		case h264util.TypePictureParameter:
			if s.cd != nil {
				s.pps = nalu
			}
		}
	}
	if s.cd != nil && len(s.pps) > 0 && len(s.sps) > 0 {
		cd, err := h264parser.NewCodecDataFromSPSAndPPS(s.sps, s.pps)
		if err != nil {
			return fmt.Errorf("parsing h264 codec data: %w", err)
		}
		s.cd <- cd
		close(s.cd)
		s.cd = nil
	}
	return nil
}
