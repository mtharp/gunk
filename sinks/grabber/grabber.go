package grabber

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"

	"eaglesong.dev/gunk/h264util"
	"eaglesong.dev/gunk/model"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/h264parser"
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
	vidIdx := -1
	var vidCodec h264parser.CodecData
	for i, s := range streams {
		if s.Type() == av.H264 {
			vidIdx = i
			vidCodec = s.(h264parser.CodecData)
		}
	}
	if vidIdx < 0 {
		return nil, errors.New("no h264 stream found")
	}
	grabch := make(chan Result, 1)
	go func() {
		defer close(grabch)
		var buf bytes.Buffer
		var keyTime time.Duration
		var lastGrab time.Time
		var lastBframe time.Duration
		for {
			pkt, err := dm.ReadPacket()
			if err == io.EOF {
				return
			} else if err != nil {
				log.Printf("error: in frame grabber: %s", err)
				return
			}
			if int(pkt.Idx) != vidIdx {
				continue
			}
			if buf.Len() != 0 && (!pkt.IsKeyFrame || pkt.Time != keyTime) {
				if time.Since(lastGrab) >= grabInterval {
					if err := makeFrame(channelName, vidCodec, buf.Bytes()); err != nil {
						log.Println("error: making thumbnail:", err)
					}
					lastGrab = time.Now()
					select {
					case grabch <- Result{
						Time:       lastGrab,
						HasBframes: lastBframe != 0,
					}:
					default:
					}
				}
				buf.Reset()
			}
			if pkt.IsKeyFrame {
				h264util.WriteAnnexBPacket(&buf, pkt, vidCodec)
				keyTime = pkt.Time
			} else {
				// check for bframes
				nalus, _ := h264parser.SplitNALUs(pkt.Data)
				for _, nalu := range nalus {
					if !h264parser.IsDataNALU(nalu) {
						continue
					}
					if sliceType, _ := h264parser.ParseSliceHeaderFromNALU(nalu); sliceType == h264parser.SLICE_B {
						lastBframe = pkt.Time
					}
				}
				if lastBframe != 0 && (pkt.Time-lastBframe) > 30*time.Second {
					lastBframe = 0
				}
			}
		}
	}()
	return grabch, nil
}

func makeFrame(channelName string, cd h264parser.CodecData, raw []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	height := targetWidth * cd.Height() / cd.Width()
	var jpeg bytes.Buffer
	var errmsg bytes.Buffer
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-loglevel", "warning",
		"-f", "h264",
		"-i", "-",
		"-frames", "1",
		"-s", fmt.Sprintf("%dx%d", targetWidth, height),
		"-f", "singlejpeg", "-")
	cmd.Stdin = bytes.NewReader(raw)
	cmd.Stdout = &jpeg
	cmd.Stderr = &errmsg
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s\n%s", err.Error(), errmsg.String())
	}
	return model.PutThumb(channelName, jpeg.Bytes())
}
