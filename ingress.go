// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"context"
	"log"
	"path"

	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/av/pktque"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/format/rtmp"
	"golang.org/x/sync/errgroup"
)

func (s *gunkServer) handleRTMP(conn *rtmp.Conn) {
	fm := &pktque.FilterDemuxer{
		Demuxer: conn,
		Filter:  &pktque.FixTime{MakeIncrement: true},
	}
	streams, err := fm.Streams()
	if err != nil {
		log.Printf("[ingress] error: reading streams on %s: %s", conn.URL, err)
		return
	}
	chname := path.Base(conn.URL.Path)
	if chname == "" {
		log.Printf("[ingress] error: invalid stream name %s", conn.URL)
		return
	}
	q := pubsub.NewQueue()
	q.WriteHeader(streams)
	hls := new(HLSPublisher)

	s.mu.Lock()
	if existing := s.channels[chname]; existing != nil {
		existing.queue.Close()
	}
	ch := &channel{q, hls}
	s.channels[chname] = ch
	s.mu.Unlock()

	log.Printf("[ingress] publish of %s started", chname)
	eg, ctx := errgroup.WithContext(context.Background())
	go func() {
		<-ctx.Done()
		q.Close()
	}()
	eg.Go(func() error {
		return hls.Publish(q.Latest())
	})
	eg.Go(func() error {
		return avutil.CopyPackets(q, fm)
	})
	if err := eg.Wait(); err != nil {
		log.Printf("[ingress] error: on stream %s: %s", conn.URL, err)
	}
	log.Printf("[ingress] publish of %s stopped", chname)
	s.mu.Lock()
	if s.channels[chname] == ch {
		delete(s.channels, chname)
	}
	s.mu.Unlock()
}
