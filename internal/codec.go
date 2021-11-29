package internal

import (
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/aacparser"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/rs/zerolog"
)

func CodecTag(cd av.CodecData, e *zerolog.Event) {
	switch cd := cd.(type) {
	case av.AudioCodecData:
		e.Stringer("samp_fmt", cd.SampleFormat())
		e.Int("samp_rate", cd.SampleRate())
		e.Stringer("ch_layout", cd.ChannelLayout())
	case av.VideoCodecData:
		e.Int("w", cd.Width())
		e.Int("h", cd.Height())
	}
	switch cd := cd.(type) {
	case h264parser.CodecData:
		e.Hex("avc1_tag", []byte{
			cd.RecordInfo.AVCProfileIndication,
			cd.RecordInfo.ProfileCompatibility,
			cd.RecordInfo.AVCLevelIndication})
		e.Hex("avc1_pps", cd.PPS())
		e.Hex("avc1_sps", cd.SPS())
	case aacparser.CodecData:
		e.Hex("aac_conf", cd.MPEG4AudioConfigBytes())
	}
}
