package ingest

import (
	"context"
	"errors"
	"sync/atomic"

	"eaglesong.dev/gunk/ingest/whip"
	"eaglesong.dev/gunk/internal"
	"eaglesong.dev/gunk/model"
	"eaglesong.dev/gunk/sinks/grabber"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/rs/zerolog/log"
)

func (m *Manager) PublishRTC(auth model.ChannelAuth, offer []byte) ([]byte, string, error) {
	name := auth.Name
	c := log.Logger.With()
	c = c.Str("channel", name)
	c = c.Str("user_id", auth.UserID)
	l := c.Logger()
	ctx := l.WithContext(context.Background())
	q := pubsub.NewQueue()
	receiver, err := whip.Receive(ctx, m.rtc, offer, q)
	if err != nil {
		return nil, "", err
	}
	// go live
	sessionID := internal.RandomID(12)
	v, _ := m.channels.LoadOrStore(name, new(channel))
	ch := v.(*channel)
	ch.name = name
	ch.mu.Lock()
	if ch.whip != nil {
		ch.whip.Close()
	}
	ch.whip = receiver
	ch.whipID = sessionID
	ch.mu.Unlock()
	p := ch.setStream(q, q, q, m.WorkDir, m.PublishMode) // TODO: AAC?
	go func() {
		<-receiver.Done()
		l.Info().Msg("stopped publishing")
		if ch.stopStream(q) && m.PublishEvent != nil {
			m.PublishEvent(auth, false, grabber.Result{})
		}
	}()
	// announce
	l.Info().Msg("started publishing")
	if m.PublishEvent != nil {
		m.PublishEvent(auth, true, grabber.Result{})
	}
	// start outputs
	// eg.Go(func() error {
	go func() {
		err := ch.copyWeb(p, q.Latest())
		if err != nil {
			l.Err(err).Msg("error in web publishing")
			// err = fmt.Errorf("web publish: %w", err)
		}
		// return err
	}()
	// grab keyframes for thumbnail
	// eg.Go(func() error {
	go func() {
		grabch, err := grabber.Grab(name, q.Oldest())
		if err != nil {
			l.Err(err).Msg("error in frame grabber")
		}
		// notify ws clients when thumbnail is updated
		for thumb := range grabch {
			ch.countWebViewers()
			if m.PublishEvent != nil {
				m.PublishEvent(auth, true, thumb)
			}
			if thumb.HasBframes {
				atomic.StoreUintptr(&ch.rtc, 0)
			} else {
				atomic.StoreUintptr(&ch.rtc, 1)
			}
		}
		// return nil
	}()
	return receiver.SDP(), sessionID, nil
}

func (m *Manager) StopRTC(auth model.ChannelAuth, sessionID string) error {
	ch := m.channel(auth.Name)
	if ch == nil {
		return ErrNoChannel
	}
	ch.mu.Lock()
	defer ch.mu.Unlock()
	if ch.whip == nil || ch.whipID != sessionID {
		return errors.New("session already closed")
	}
	ch.whip.Close()
	ch.whip = nil
	return nil
}
