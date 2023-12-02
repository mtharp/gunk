package h264util

import (
	"bytes"
	"encoding/binary"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/h264parser"
)

// WriteAnnexB writes one or more NALUs with an Annex B header
func WriteAnnexB(w *bytes.Buffer, nalus [][]byte) {
	for _, nalu := range nalus {
		w.Write([]byte{0, 0, 1})
		w.Write(nalu)
	}
}

// WriteAnnexBPacket writes a H264 packet in Annex B format.
// If the packet is a keyframe, prepend codec information (SPS and PPS) as well.
func WriteAnnexBPacket(w *bytes.Buffer, pkt av.Packet, cd h264parser.CodecData) {
	nalus, _ := h264parser.SplitNALUs(pkt.Data)
	if pkt.IsKeyFrame {
		nalus = append([][]byte{cd.SPS(), cd.PPS()}, nalus...)
	}
	WriteAnnexB(w, nalus)
}

// NALUToAVCC converts a raw NALU to AVCC format
func NALUToAVCC(nalu []byte) []byte {
	b := make([]byte, 4+len(nalu))
	binary.BigEndian.PutUint32(b, uint32(len(nalu)))
	copy(b[4:], nalu)
	return b
}
