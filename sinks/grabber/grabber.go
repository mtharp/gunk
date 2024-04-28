package grabber

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"eaglesong.dev/gunk/h264util"
	"eaglesong.dev/gunk/model"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/av1parser"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/nareix/joy4/codec/h265parser"
	"github.com/rs/zerolog/log"
)

const (
	targetWidth  = 400
	grabInterval = 10 * time.Second
)

type Result struct {
	Time       time.Time
	HasBframes bool
}

func Grab(channelName string, dm av.Demuxer) (<-chan Result, error) {
	streams, err := dm.Streams()
	if err != nil {
		return nil, err
	}
	var pushKeyframe func(*bytes.Buffer, av.Packet)
	var makeFrame func(context.Context, *bytes.Buffer) ([]byte, error)
	hasBframes := func(av.Packet) bool { return false }
	var vidIdx int8 = -1
	for i, s := range streams {
		if !s.Type().IsVideo() {
			continue
		}
		vidIdx = int8(i)
		switch cd := s.(type) {
		case h264parser.CodecData:
			pushKeyframe = func(buf *bytes.Buffer, pkt av.Packet) {
				h264util.WriteAnnexBPacket(buf, pkt, cd)
			}
			makeFrame = func(ctx context.Context, b *bytes.Buffer) ([]byte, error) {
				return ffmpegThumbnail(ctx, cd, b.Bytes(), "h264")
			}
			hasBframes = func(pkt av.Packet) bool {
				nalus, _ := h264parser.SplitNALUs(pkt.Data)
				for _, nalu := range nalus {
					if !h264parser.IsDataNALU(nalu) {
						continue
					}
					if sliceType, _ := h264parser.ParseSliceHeaderFromNALU(nalu); sliceType == h264parser.SLICE_B {
						return true
					}
				}
				return false
			}
		case h265parser.CodecData:
			pushKeyframe = func(buf *bytes.Buffer, pkt av.Packet) {
				h265parser.WriteAnnexBPacket(buf, pkt, cd)
			}
			makeFrame = func(ctx context.Context, b *bytes.Buffer) ([]byte, error) {
				return ffmpegThumbnail(ctx, cd, b.Bytes(), "hevc")
			}
		case av1parser.CodecData:
			pushKeyframe = func(b *bytes.Buffer, p av.Packet) {
				// TODO - is reformatting needed?
				b.Write(av1parser.TemporalDelimiter)
				b.Write(cd.ConfigOBUs)
				b.Write(p.Data)
			}
			makeFrame = func(ctx context.Context, b *bytes.Buffer) ([]byte, error) {
				return ffmpegThumbnail(ctx, cd, b.Bytes(), "obu")
			}
		default:
			return nil, fmt.Errorf("grabber: unsupported video codec %s", s.Type().String())
		}
	}
	if vidIdx < 0 {
		return nil, fmt.Errorf("no video found")
	}
	grabch := make(chan Result, 1)
	go func() {
		defer close(grabch)
		var buf bytes.Buffer
		var keyTime time.Duration
		var lastGrab time.Time
		var lastBframe time.Duration
		l := log.With().Str("channel", channelName).Logger()
		for {
			pkt, err := dm.ReadPacket()
			if err == io.EOF {
				return
			} else if err != nil {
				l.Err(err).Msg("failed to grab frame")
				return
			}
			if pkt.Idx != vidIdx {
				continue
			}
			if buf.Len() != 0 && (!pkt.IsKeyFrame || pkt.Time != keyTime) {
				if time.Since(lastGrab) >= grabInterval {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					jpegBytes, err := makeFrame(ctx, &buf)
					if err == nil {
						err = model.PutThumb(ctx, channelName, jpegBytes)
					}
					cancel()
					if err != nil {
						l.Err(err).Msg("failed to make thumbnail")
					} else {
						lastGrab = time.Now()
						select {
						case grabch <- Result{
							Time:       lastGrab,
							HasBframes: lastBframe != 0,
						}:
						default:
						}
					}
				}
				buf.Reset()
			}
			if pkt.IsKeyFrame {
				if buf.Len() != 0 {
					// don't push parameter sets more than once
					pkt.IsKeyFrame = false
				}
				pushKeyframe(&buf, pkt)
				keyTime = pkt.Time
			} else {
				// check for bframes
				if hasBframes(pkt) {
					lastBframe = pkt.Time
				}
				if lastBframe != 0 && (pkt.Time-lastBframe) > 30*time.Second {
					lastBframe = 0
				}
			}
		}
	}()
	return grabch, nil
}

func ffmpegThumbnail(ctx context.Context, cd av.VideoCodecData, raw []byte, format string) ([]byte, error) {
	height := targetWidth * cd.Height() / cd.Width()
	var jpeg bytes.Buffer
	var errmsg bytes.Buffer
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-loglevel", "warning",
		"-f", format,
		"-i", "-",
		"-frames:v", "1",
		"-s", fmt.Sprintf("%dx%d", targetWidth, height),
		"-f", "mjpeg", "-")
	cmd.Stdin = bytes.NewReader(raw)
	cmd.Stdout = &jpeg
	cmd.Stderr = &errmsg
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s\n%s", err.Error(), errmsg.String())
	}
	return jpeg.Bytes(), nil
}
