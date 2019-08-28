package web

import (
	"context"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"eaglesong.dev/gunk/model"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var wsu = websocket.Upgrader{HandshakeTimeout: 10 * time.Second}

type listener chan<- wsMsg

type websockets struct {
	mu        sync.Mutex
	listeners map[listener]struct{}

	OnNew func(*websocket.Conn) error
}

type wsMsg struct {
	Type    string             `json:"type"`
	Channel *model.ChannelInfo `json:"channel,omitempty"`
}

func channelWS(i *model.ChannelInfo) wsMsg {
	return wsMsg{Type: "channel", Channel: i}
}

func (w *websockets) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	conn, err := wsu.Upgrade(rw, req, nil)
	if err != nil {
		log.Println("error: websocket upgrade:", err)
		return
	}
	eg, ctx := errgroup.WithContext(req.Context())
	eg.Go(func() error { return w.drainLoop(ctx, conn) })
	eg.Go(func() error { return w.sendLoop(ctx, conn) })
	eg.Go(func() error {
		<-ctx.Done()
		conn.Close()
		return nil
	})
	if err := eg.Wait(); err != nil && err != io.EOF {
		log.Printf("error: websocket %s: %s", conn.RemoteAddr(), err)
	}
}

func (w *websockets) Broadcast(msg wsMsg) {
	w.mu.Lock()
	for listener := range w.listeners {
		select {
		case listener <- msg:
		default:
			// on overflow force the client to reconnect
			delete(w.listeners, listener)
			close(listener)
		}
	}
	w.mu.Unlock()
}

func (w *websockets) drainLoop(ctx context.Context, conn *websocket.Conn) error {
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

func (w *websockets) sendLoop(ctx context.Context, conn *websocket.Conn) error {
	ch := make(chan wsMsg, 10)
	w.mu.Lock()
	if w.listeners == nil {
		w.listeners = make(map[listener]struct{})
	}
	w.listeners[ch] = struct{}{}
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		delete(w.listeners, ch)
		w.mu.Unlock()
	}()
	if w.OnNew != nil {
		if err := w.OnNew(conn); err != nil {
			return err
		}
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
