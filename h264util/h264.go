// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package h264util

import (
	"bytes"

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
// If the packet is a keyframe, prepend codec information (SPS and PPS) as well.
func WriteAnnexBPacket(w *bytes.Buffer, pkt av.Packet, cd h264parser.CodecData) {
	nalus, _ := h264parser.SplitNALUs(pkt.Data)
	if pkt.IsKeyFrame {
		nalus = append(nalus, cd.SPS(), cd.PPS())
	}
	WriteAnnexB(w, nalus)
}
