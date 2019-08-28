package web

import (
	"log"
	"time"

	"eaglesong.dev/gunk/model"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

func (s *Server) PublishEvent(auth model.ChannelAuth, live bool, thumbTime time.Time) {
	if live && thumbTime.IsZero() {
		go func() {
			if err := s.doWebhook(auth); err != nil {
				log.Printf("warning: invoking webhook for %s: %s", auth.Name, err)
			}
		}()
	}
	if !live || !thumbTime.IsZero() {
		ch := &model.ChannelInfo{
			Name: auth.Name,
			Live: live,
			Last: thumbTime.UnixNano() / 1000000,
		}
		s.populateChannel(ch)
		s.ws.Broadcast(channelWS(ch))
	}
}

func (s *Server) onWebsocket(conn *websocket.Conn) error {
	channels, err := s.listChannels()
	if err != nil {
		return errors.Wrap(err, "listing channels")
	}
	for _, channel := range channels {
		if err := conn.WriteJSON(channelWS(channel)); err != nil {
			return errors.Wrap(err, "write")
		}
	}
	return nil
}

func (s *Server) wsChannelLive(name string, live bool, thumb time.Time) error {

	return nil
}
