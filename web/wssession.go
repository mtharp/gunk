package web

import (
	"sync"
	"time"

	"eaglesong.dev/gunk/internal"
	"eaglesong.dev/gunk/sinks/playrtc"
)

const wsSessionTimeout = time.Minute

type wsSession struct {
	server *Server
	ID     string
	send   chan wsMsg
	// protected by Server.smu
	conn     *wsConn
	deadline time.Time
	// protected by mu
	mu  sync.Mutex
	rtc *playrtc.Sender
}

func (s *Server) newSession(conn *wsConn, resume string) *wsSession {
	s.smu.Lock()
	defer s.smu.Unlock()
	var n *wsSession
	if resume != "" {
		n = s.sessions[resume]
	}
	if n == nil {
		// new session
		n = &wsSession{
			server: s,
			ID:     internal.RandomID(16),
			send:   make(chan wsMsg, 10),
		}
		s.sessions[n.ID] = n
		// log.Println("[ws] created", n.ID)
	} else if n.conn != nil {
		// terminate old connection
		n.conn.cancel()
		// 	log.Println("[ws] replacing", n.ID)
		// } else {
		// 	log.Println("[ws] resumed", n.ID)
	}
	conn.session = n
	n.conn = conn
	n.deadline = time.Time{}
	return n
}

func (s *Server) wsDisconnected(n *wsSession, conn *wsConn) {
	s.smu.Lock()
	if n.conn == conn {
		n.conn = nil
		n.deadline = time.Now().Add(wsSessionTimeout)
	}
	s.smu.Unlock()
	// log.Println("[ws] disconnected", n.ID)
}

func (n *wsSession) close() {
	n.mu.Lock()
	defer n.mu.Unlock()
	// log.Println("[ws] released", n.ID)
	if n.rtc != nil {
		n.rtc.Close()
		n.rtc = nil
	}
}

func (s *Server) checkSessions() {
	t := time.NewTicker(10 * time.Second)
	for range t.C {
		s.smu.Lock()
		var dead []*wsSession
		for id, n := range s.sessions {
			if n.deadline.IsZero() || time.Until(n.deadline) > 0 {
				continue
			}
			dead = append(dead, n)
			delete(s.sessions, id)
		}
		s.smu.Unlock()
		for _, n := range dead {
			n.close()
		}
	}
}
