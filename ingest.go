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
	defer conn.Close()
	remote := conn.NetConn().RemoteAddr()
	fm := &pktque.FilterDemuxer{
		Demuxer: conn,
		Filter:  &pktque.FixTime{MakeIncrement: true},
	}
	streams, err := fm.Streams()
	if err != nil {
		log.Printf("[ingest] error: reading streams on %s from %s: %s", conn.URL, remote, err)
		return
	}
	userID := verifyChannel(conn.URL)
	if userID == "" {
		log.Printf("[ingest] error: stream not found or incorrect key for %s from %s", conn.URL, remote)
		return
	}
	chname := path.Base(conn.URL.Path)
	q := pubsub.NewQueue()
	q.WriteHeader(streams)
	hls := new(HLSPublisher)
	if err := grabFrames(chname, q.Latest()); err != nil {
		log.Printf("[ingest] error: setting up frame grabber on %s from %s: %s", conn.URL, remote, err)
		return
	}

	s.mu.Lock()
	if existing := s.channels[chname]; existing != nil {
		existing.queue.Close()
	}
	ch := &channel{q, hls}
	s.channels[chname] = ch
	s.mu.Unlock()

	log.Printf("[ingest] publish of %s started from %s on behalf of user %s", chname, remote, userID)
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
		log.Printf("[ingest] error: on stream %s from %s: %s", conn.URL, remote, err)
	}
	log.Printf("[ingest] publish of %s stopped from %s on behalf of user %s", chname, remote, userID)
	s.mu.Lock()
	if s.channels[chname] == ch {
		delete(s.channels, chname)
	}
	s.mu.Unlock()
}
