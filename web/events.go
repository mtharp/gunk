package web

import (
	"log"
	"time"

	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/sinks/grabber"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

func (s *Server) PublishEvent(auth model.ChannelAuth, live bool, thumb grabber.Result) {
	if live && thumb.Time.IsZero() {
		go func() {
			if err := s.doWebhook(auth); err != nil {
				log.Printf("warning: invoking webhook for %s: %s", auth.Name, err)
			}
		}()
	}
	if !live || !thumb.Time.IsZero() {
		ch := &model.ChannelInfo{
			Name: auth.Name,
			Live: live,
			Last: thumb.Time.UnixNano() / 1000000,
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
