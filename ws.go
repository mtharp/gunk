// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var wsu = websocket.Upgrader{HandshakeTimeout: 10 * time.Second}

type listener chan<- wsMsg

type wsMsg struct {
	Type    string       `json:"type"`
	Channel *channelInfo `json:"channel,omitempty"`
}

func (i channelInfo) asMessage() wsMsg {
	return wsMsg{Type: "channel", Channel: &i}
}

func (s *gunkServer) handleWS(rw http.ResponseWriter, req *http.Request) {
	conn, err := wsu.Upgrade(rw, req, nil)
	if err != nil {
		log.Println("error: websocket upgrade:", err)
		return
	}
	eg, ctx := errgroup.WithContext(req.Context())
	eg.Go(func() error { return s.wsDrainLoop(ctx, conn) })
	eg.Go(func() error { return s.wsSendLoop(ctx, conn) })
	eg.Go(func() error {
		<-ctx.Done()
		conn.Close()
		return nil
	})
	if err := eg.Wait(); err != nil && err != io.EOF {
		log.Printf("error: websocket %s: %s", conn.RemoteAddr(), err)
	}
}

func (s *gunkServer) wsDrainLoop(ctx context.Context, conn *websocket.Conn) error {
	for ctx.Err() == nil {
		if _, _, err := conn.NextReader(); err != nil {
			if ctx.Err() == nil && !websocket.IsCloseError(err, websocket.CloseGoingAway) {
				return errors.Wrap(err, "read")
			}
			break
		}
	}
	return io.EOF
}

func (s *gunkServer) wsSendLoop(ctx context.Context, conn *websocket.Conn) error {
	ch := make(chan wsMsg, 10)
	s.mu.Lock()
	s.listeners[ch] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.listeners, ch)
		s.mu.Unlock()
	}()
	if err := s.wsSendInitial(conn); err != nil {
		return err
	}
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				// channel closed after a prior send overflowed
				return io.EOF
			}
			if err := conn.WriteJSON(msg); err != nil {
				return errors.Wrap(err, "write")
			}
		}
	}
	return nil
}

func (s *gunkServer) wsSendInitial(conn *websocket.Conn) error {
	channels, err := s.listChannels()
	if err != nil {
		return errors.Wrap(err, "listing channels")
	}
	for _, channel := range channels {
		if err := conn.WriteJSON(channel.asMessage()); err != nil {
			return errors.Wrap(err, "write")
		}
	}
	return nil
}

func (s *gunkServer) wsBroadcast(msg wsMsg) {
	s.mu.Lock()
	for listener := range s.listeners {
		select {
		case listener <- msg:
		default:
			// on overflow force the client to reconnect
			delete(s.listeners, listener)
			close(listener)
		}
	}
	s.mu.Unlock()
}

func (s *gunkServer) wsChannelLive(name string, live bool, thumb time.Time) error {
	ch := &channelInfo{
		Name: name,
		Live: live,
		Last: thumb.UnixNano() / 1000000,
	}
	if err := s.populateChannel(ch); err != nil {
		return err
	}
	s.wsBroadcast(ch.asMessage())
	return nil
}
