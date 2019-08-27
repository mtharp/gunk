// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"context"
	"io"
	"log"
	"net"
	"time"

	"eaglesong.dev/gunk/opus"
	"eaglesong.dev/hls"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/av/pktque"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/format/rtmp"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

func (s *gunkServer) handleRTMP(conn *rtmp.Conn) {
	defer conn.Close()
	remote := conn.NetConn().RemoteAddr().(*net.TCPAddr).IP.String()
	fm := &pktque.FilterDemuxer{
		Demuxer: conn,
		Filter:  &pktque.FixTime{MakeIncrement: true},
	}
	auth, err := verifyRTMP(conn.URL)
	if err != nil {
		log.Printf("[rtmp] error: %s from %s: %s", conn.URL, remote, err)
		return
	}
	if err := s.handleIngest(auth, "rtmp", remote, fm); err != nil {
		log.Printf("[rtmp] error: %s from %s: %s", conn.URL, remote, err)
	}
}

func (s *gunkServer) handleIngest(authI interface{}, kind, remote string, src av.Demuxer) error {
	auth := authI.(channelAuth)
	streams, err := src.Streams()
	if err != nil {
		return errors.Wrap(err, "reading streams")
	}
	q := pubsub.NewQueue()
	q.WriteHeader(streams)
	eg, ctx := errgroup.WithContext(context.Background())
	go func() {
		<-ctx.Done()
		q.Close()
	}()
	// grab keyframes for thumbnail
	grabch, err := grabFrames(auth.Name, q.Latest())
	if err != nil {
		return errors.Wrap(err, "setting up frame grabber")
	}
	opusq := q
	if audioType(streams) != opus.OPUS {
		// convert audio to opus for webrtc
		opusq = pubsub.NewQueue()
		eg.Go(func() error {
			defer opusq.Close()
			return errors.Wrap(opus.Convert(q.Latest(), opusq, s.opusBitrate), "opus conversion")
		})
	}

	// go live
	ch := &channel{q, opusq}
	s.mu.Lock()
	if existing := s.channels[auth.Name]; existing != nil {
		existing.queue.Close()
	}
	s.channels[auth.Name] = ch
	p := s.hls[auth.Name]
	if p != nil {
		// stream restarted so viewer should reset their decoder
		p.Discontinuity()
	} else {
		p = new(hls.Publisher)
		s.hls[auth.Name] = p
	}
	s.mu.Unlock()
	defer func() {
		log.Printf("[%s] publish of %s stopped", kind, auth.Name)
		s.mu.Lock()
		if s.channels[auth.Name] == ch {
			delete(s.channels, auth.Name)
		}
		s.mu.Unlock()
		if err := s.wsChannelLive(auth.Name, false, time.Time{}); err != nil {
			log.Printf("[%s] warning: %s: %s", kind, auth.Name, err)
		}
	}()
	// announce
	log.Printf("[%s] user %s started publishing to %s from %s", kind, auth.UserID, auth.Name, remote)
	if auth.Announce && s.webhookURL != "" {
		go func() {
			if err := s.doWebhook(auth); err != nil {
				log.Printf("warning: failed to trigger webhook: %s", err)
			}
		}()
	}

	// start outputs
	eg.Go(func() error {
		return errors.Wrap(avutil.CopyFile(p, q.Latest()), "hls publish")
	})
	eg.Go(func() error {
		defer q.Close()
		for ctx.Err() == nil {
			pkt, err := src.ReadPacket()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			if err := q.WritePacket(pkt); err != nil {
				return err
			}
		}
		return nil
	})
	eg.Go(func() error {
		// notify ws clients when thumbnail is updated
		for thumbTime := range grabch {
			if err := s.wsChannelLive(auth.Name, true, thumbTime); err != nil {
				log.Println("warning:", err)
			}
		}
		return nil
	})
	return eg.Wait()
}

func noeof(err error) error {
	if err == io.EOF {
		return nil
	}
	return err
}

func audioType(streams []av.CodecData) av.CodecType {
	for _, stream := range streams {
		if stream.Type().IsAudio() {
			return stream.Type()
		}
	}
	return 0
}
