package web

import (
	"log"

	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/sinks/grabber"
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
	}
}
