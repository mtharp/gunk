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
}

type channelMarkers map[string]*model.ChannelInfo

func (m channelMarkers) Filter(infos []*model.ChannelInfo) (ret []*model.ChannelInfo) {
	for _, ch := range infos {
		if ch.Equal(m[ch.Name]) {
			continue
		}
		ret = append(ret, ch)
		m[ch.Name] = ch
	}
	return
}
