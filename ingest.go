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

	"github.com/mtharp/gunk/opus"
	"github.com/nareix/joy4/av"
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
	userID, chname, err := verifyRTMP(conn.URL)
	if err != nil {
		log.Printf("[rtmp] error: %s from %s: %s", conn.URL, remote, err)
		return
	}
	log.Printf("[rtmp] user %s started publishing to %s from %s", userID, chname, remote)
	if err := s.handleIngest(chname, userID, remote, fm); err != nil {
		log.Printf("[rtmp] error: %s from %s: %s", conn.URL, remote, err)
	}
	log.Printf("[rtmp] publish of %s stopped", chname)
}

func (s *gunkServer) handleIngest(chname, userID, remote string, src av.Demuxer) error {
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
	grabch, err := grabFrames(chname, q.Latest())
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
	hls := new(HLSPublisher)
	s.mu.Lock()
	if existing := s.channels[chname]; existing != nil {
		existing.queue.Close()
	}
	ch := &channel{q, opusq, hls}
	s.channels[chname] = ch
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		if s.channels[chname] == ch {
			delete(s.channels, chname)
		}
		s.mu.Unlock()
		if err := s.wsChannelLive(chname, false, time.Time{}); err != nil {
			log.Printf("[ingest] warning: %s: %s", chname, err)
		}
	}()

	// start outputs
	eg.Go(func() error {
		return errors.Wrap(hls.Publish(q.Latest()), "hls publish")
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
			if err := s.wsChannelLive(chname, true, thumbTime); err != nil {
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
