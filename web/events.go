package web

import (
	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/sinks/grabber"
	"github.com/rs/zerolog/log"
)

func (s *Server) PublishEvent(auth model.ChannelAuth, live bool, thumb grabber.Result) {
	if live && thumb.Time.IsZero() {
		go func() {
			if err := s.doWebhook(auth); err != nil {
				log.Err(err).Str("channel", auth.Name).Msg("failed to invoke webhook")
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
