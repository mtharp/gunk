package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"time"

	"eaglesong.dev/gunk/model"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"golang.org/x/sync/errgroup"
)

var wsu = websocket.Upgrader{HandshakeTimeout: 10 * time.Second}

type wsConn struct {
	server  *Server
	session *wsSession
	conn    *websocket.Conn
	cancel  context.CancelFunc
}

type wsMsg struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`

	SDP       *webrtc.SessionDescription `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
	Channel   *model.ChannelInfo         `json:"channel,omitempty"`
}

func (s *Server) serveWS(rw http.ResponseWriter, req *http.Request) {
	conn, err := wsu.Upgrade(rw, req, nil)
	if err != nil {
		log.Println("error: websocket upgrade:", err)
		return
	}
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	w := &wsConn{
		server: s,
		conn:   conn,
		cancel: cancel,
	}
	var n *wsSession
	resume := req.URL.Query().Get("session")
	n = s.newSession(w, resume)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error { return w.recvLoop(ctx) })
	eg.Go(func() error { return w.sendLoop(ctx) })
	eg.Go(func() error {
		<-ctx.Done()
		s.wsDisconnected(n, w)
		conn.Close()
		return nil

	})
	n.send <- wsMsg{
		Type: "connected",
		ID:   n.ID,
	}
	if err := eg.Wait(); err != nil && err != io.EOF {
		log.Printf("error: websocket %s: %s", conn.RemoteAddr(), err)
	}
}

func (w *wsConn) recvLoop(ctx context.Context) (err error) {
	defer func() {
		if err2 := recover(); err2 != nil {
			buf := make([]byte, 1e5)
			buf = buf[:runtime.Stack(buf, false)]
			log.Printf("uncaught panic in handler: %+v\n%s", err2, string(buf))
			if err != nil {
				err = errors.New("panic caught")
			}
		}
	}()
	for ctx.Err() == nil {
		var m wsMsg
		if err := w.conn.ReadJSON(&m); err != nil {
			if ctx.Err() == nil && !websocket.IsCloseError(err, websocket.CloseGoingAway) {
				return fmt.Errorf("read: %w", err)
			}
			break
		}
		if err := w.handle(m); err != nil {
			return err
		}
	}
	return io.EOF
}

func (w *wsConn) sendLoop(ctx context.Context) (err error) {
	defer func() {
		if err2 := recover(); err2 != nil {
			buf := make([]byte, 1e5)
			buf = buf[:runtime.Stack(buf, false)]
			log.Printf("uncaught panic in handler: %+v\n%s", err2, string(buf))
			if err != nil {
				err = errors.New("panic caught")
			}
		}
	}()
	markers := make(channelMarkers)
	t := time.NewTicker(2 * time.Second)
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-w.session.send:
			if !ok {
				return io.EOF
			}
			if err := w.conn.WriteJSON(msg); err != nil {
				return fmt.Errorf("write: %w", err)
			}
		case <-t.C:
			changed, err := w.server.listChannels(markers)
			if err != nil {
				log.Println("error: listing channels:", err)
			}
			for _, ch := range changed {
				msg := wsMsg{Type: "channel", Channel: ch}
				if err := w.conn.WriteJSON(msg); err != nil {
					return fmt.Errorf("write: %w", err)
				}
			}
			if len(changed) == 0 {
				msg := wsMsg{Type: "idle"}
				if err := w.conn.WriteJSON(msg); err != nil {
					return fmt.Errorf("write: %w", err)
				}
			}
		}
	}
	return nil
}

func (w *wsConn) handle(m wsMsg) error {
	switch m.Type {
	case "play":
		return w.session.Play(m.Name, w.conn.RemoteAddr().String())
	case "candidate":
		return w.session.Candidate(m.Candidate)
	case "answer":
		return w.session.Answer(m.SDP)
	case "stop":
		return w.session.Stop()
	case "ping":
		return nil
	default:
		return errors.New("invalid message type " + m.Type)
	}
}
