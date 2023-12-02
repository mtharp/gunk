package whip

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"eaglesong.dev/gunk/internal"
	"eaglesong.dev/gunk/internal/rtcengine"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/codec/opusparser"
	"github.com/pion/interceptor/pkg/stats"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
	"github.com/rs/zerolog"
)

const (
	videoIdx int8 = iota
	audioIdx
)

const (
	maxLateSeq  = 1000
	maxLateTime = 5 * time.Second
)

type Receiver struct {
	pc     *webrtc.PeerConnection
	stats  stats.Getter
	state  uintptr
	ctx    context.Context
	cancel context.CancelFunc
	log    *zerolog.Logger
	sdp    webrtc.SessionDescription
}

func Receive(ctx context.Context, e *rtcengine.Engine, offer []byte, dest *pubsub.Queue) (*Receiver, error) {
	pc, stats, err := e.Connection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	r := &Receiver{
		pc:     pc,
		stats:  stats,
		ctx:    ctx,
		cancel: cancel,
		log:    zerolog.Ctx(ctx),
	}
	return r, r.start(offer, dest)
}

func (r *Receiver) start(offer []byte, dest *pubsub.Queue) error {
	r.log.Debug().Str("rtc_offer_received", string(offer)).Send()
	// setup callbacks
	gatherWait, gatherDone := context.WithCancel(context.Background())
	r.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		atomic.StoreUintptr(&r.state, uintptr(state))
		r.log.Info().Stringer("rtc_state", state).Send()
		switch state {
		case webrtc.ICEConnectionStateDisconnected, webrtc.ICEConnectionStateFailed, webrtc.ICEConnectionStateClosed:
			r.Close()
			dest.Close()
		}
	})
	r.pc.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		r.log.Info().Stringer("ice_state", state).Send()
		if state == webrtc.ICEGathererStateComplete {
			// release the SDP answer when all candidates are ready
			gatherDone()
		}
	})
	codecDatas := r.startGatheringHeaders(dest)
	r.pc.OnTrack(func(tr *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		switch strings.ToLower(tr.Codec().MimeType) {
		case strings.ToLower(webrtc.MimeTypeOpus):
			r.startOpusTrack(tr, dest, codecDatas[audioIdx])
		case strings.ToLower(webrtc.MimeTypeH264):
			r.startH264Track(tr, dest, codecDatas[videoIdx])
		default:
			r.log.Error().Msgf("unsupported codec type %s", tr.Codec().MimeType)
			return
		}
		r.log.Debug().Msgf("new incoming track: %s", tr.Codec().MimeType)
	})
	// create answer
	if err := r.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  string(offer),
	}); err != nil {
		return fmt.Errorf("setting remote description: %w", err)
	}
	initialAnswer, err := r.pc.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("creating answer: %w", err)
	}
	if err := r.pc.SetLocalDescription(initialAnswer); err != nil {
		return fmt.Errorf("setting local description: %w", err)
	}
	// Wait for ICE gathering to complete, as we have no way to trickle.
	// In practice this should be instant, as our endpoint is a static port.
	select {
	case <-r.ctx.Done():
		r.Close()
		return r.ctx.Err()
	case <-gatherWait.Done():
	}
	finalAnswer := r.pc.LocalDescription()
	if finalAnswer == nil {
		r.Close()
		return errors.New("local description is unset")
	}
	r.sdp = *finalAnswer
	r.log.Debug().Str("rtp_answer_sent", r.sdp.SDP).Send()
	return nil
}

func (r *Receiver) SDP() []byte {
	return []byte(r.sdp.SDP)
}

func (r *Receiver) Done() <-chan struct{} {
	return r.ctx.Done()
}

func (r *Receiver) Close() {
	r.cancel()
	r.pc.Close()
}

func (r *Receiver) receiveTrack(tr *webrtc.TrackRemote, dest av.PacketWriter, idx int8, depacketizer rtp.Depacketizer, sampler sampler) error {
	var lastKey time.Duration
	builder := samplebuilder.New(maxLateSeq, depacketizer, tr.Codec().ClockRate, samplebuilder.WithMaxTimeDelay(maxLateTime))
	timescale := internal.RelativeConverter{Rate: uint64(tr.Codec().ClockRate)}
	for {
		pkt, _, err := tr.ReadRTP()
		if err != nil {
			return err
		}
		builder.Push(pkt)
		for {
			sample := builder.Pop()
			if sample == nil {
				break
			}
			pkt := av.Packet{
				Idx:  idx,
				Time: timescale.Convert(sample.PacketTimestamp),
				Data: sample.Data,
			}
			if sampler != nil {
				if err := sampler.Update(&pkt); err != nil {
					return err
				}
			}
			// Avoid back-to-back IDR slices both reporting as keyframes
			if pkt.Time == lastKey {
				pkt.IsKeyFrame = false
			} else if pkt.IsKeyFrame {
				lastKey = pkt.Time
			}
			dest.WritePacket(pkt)
		}
	}
}

func (r *Receiver) trackStats(tr *webrtc.TrackRemote) {
	if r.stats == nil {
		return
	}
	statLog := r.log.With().
		Str("mtype", tr.Codec().MimeType).
		Uint32("ssrc", uint32(tr.SSRC())).
		Logger()
	ssrc := uint32(tr.SSRC())
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-t.C:
			st := r.stats.Get(ssrc).InboundRTPStreamStats
			statLog.Info().
				Uint64("rx_packets", st.PacketsReceived).
				Uint64("rx_bytes", st.BytesReceived).
				Int64("lost_packets", st.PacketsLost).
				Float64("jitter", st.Jitter).
				Uint32("fir_count", st.FIRCount).
				Uint32("pli_count", st.PLICount).
				Uint32("nack_count", st.NACKCount).
				Dur("time_delta", time.Since(st.LastPacketReceivedTimestamp)).
				Msg("WHIP receiver stats")
		}
	}
}

func (r *Receiver) startOpusTrack(tr *webrtc.TrackRemote, dest av.PacketWriter, codecData chan<- av.CodecData) {
	codecData <- &opusparser.CodecData{
		Channels: int(tr.Codec().Channels),
	}
	go r.trackStats(tr)
	go func() {
		if err := r.receiveTrack(tr, dest, audioIdx, &codecs.OpusPacket{}, nil); err != nil {
			r.log.Err(err).Msg("opus track terminated")
		}
	}()
}

func (r *Receiver) startH264Track(tr *webrtc.TrackRemote, dest av.PacketWriter, codecData chan<- av.CodecData) {
	go r.trackStats(tr)
	go func() {
		dpkt := &codecs.H264Packet{IsAVC: true} // convert to AVCC
		sampler := &h264sampler{cd: codecData}
		if err := r.receiveTrack(tr, dest, videoIdx, dpkt, sampler); err != nil {
			r.log.Err(err).Msg("h264 track terminated")
		}
	}()
}

func (r *Receiver) startGatheringHeaders(dest av.Muxer) []chan av.CodecData {
	// make channels to receive each track's CodecData
	channels := []chan av.CodecData{
		make(chan av.CodecData, 1),
		make(chan av.CodecData, 1),
	}
	go func() {
		// wait for all CodecData to be available
		streams := make([]av.CodecData, 2)
		for i, channel := range channels {
			select {
			case cd := <-channel:
				if cd == nil {
					r.log.Error().Msgf("failed to set codecdata: idx %d returned nil", i)
					return
				}
				streams[i] = cd
			case <-r.ctx.Done():
				return
			}
		}
		// write header all at once
		if err := dest.WriteHeader(streams); err != nil {
			r.log.Err(err).Msg("failed to set codecdata")
			r.Close()
		}
	}()
	return channels
}
