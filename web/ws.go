package web

// var wsu = websocket.Upgrader{HandshakeTimeout: 10 * time.Second}

// type wsConn struct {
// 	conn  *websocket.Conn
// 	rtc   *playrtc.Sender
// 	send  chan wsMsg
// 	chans *ingest.Manager
// }

// type wsMsg struct {
// 	Type string `json:"type"`
// 	Name string `json:"name,omitempty"`
// 	Time int64  `json:"time,omitempty"`

// 	SDP       *webrtc.SessionDescription `json:"sdp,omitempty"`
// 	Candidate *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
// }

// func (s *Server) serveWS(rw http.ResponseWriter, req *http.Request) {
// 	conn, err := wsu.Upgrade(rw, req, nil)
// 	if err != nil {
// 		log.Println("error: websocket upgrade:", err)
// 		return
// 	}
// 	w := &wsConn{
// 		conn:  conn,
// 		send:  make(chan wsMsg, 10),
// 		chans: &s.Channels,
// 	}
// 	ctx, cancel := context.WithCancel(req.Context())
// 	defer cancel()
// 	eg, ctx := errgroup.WithContext(ctx)
// 	eg.Go(func() error { return w.recvLoop(ctx) })
// 	eg.Go(func() error { return w.sendLoop(ctx) })
// 	eg.Go(func() error {
// 		<-ctx.Done()
// 		conn.Close()
// 		if w.rtc != nil {
// 			w.rtc.Close()
// 			w.rtc = nil
// 		}
// 		return nil
// 	})
// 	if err := eg.Wait(); err != nil && err != io.EOF {
// 		log.Printf("error: websocket %s: %s", conn.RemoteAddr(), err)
// 	}
// }

// func (w *wsConn) recvLoop(ctx context.Context) error {
// 	for ctx.Err() == nil {
// 		var m wsMsg
// 		if err := w.conn.ReadJSON(&m); err != nil {
// 			if ctx.Err() == nil && !websocket.IsCloseError(err, websocket.CloseGoingAway) {
// 				return fmt.Errorf("read: %w", err)
// 			}
// 			break
// 		}
// 		if err := w.handle(m); err != nil {
// 			return err
// 		}
// 	}
// 	return io.EOF
// }

// func (w *wsConn) sendLoop(ctx context.Context) error {
// 	t := time.NewTicker(time.Second)
// 	for ctx.Err() == nil {
// 		select {
// 		case <-ctx.Done():
// 			return nil
// 		case msg, ok := <-w.send:
// 			if !ok {
// 				return io.EOF
// 			}
// 			if err := w.conn.WriteJSON(msg); err != nil {
// 				return fmt.Errorf("write: %w", err)
// 			}
// 		case <-t.C:
// 			msg := wsMsg{Type: "ts", Time: time.Now().UnixNano() / 1000000}
// 			if err := w.conn.WriteJSON(msg); err != nil {
// 				return fmt.Errorf("write: %w", err)
// 			}
// 		}
// 	}
// 	return nil
// }

// func (w *wsConn) handle(m wsMsg) error {
// 	switch m.Type {
// 	case "offer":
// 		if w.rtc != nil {
// 			w.rtc.Close()
// 			w.rtc = nil
// 		}
// 		if m.SDP == nil {
// 			return errors.New("missing sdp")
// 		}
// 		o := playrtc.OfferToReceive{Offer: *m.SDP}
// 		o.Remote = w.conn.RemoteAddr().String()
// 		o.SendCandidate = func(cand webrtc.ICECandidateInit) {
// 			w.send <- wsMsg{
// 				Type:      "candidate",
// 				Candidate: &cand,
// 			}
// 		}
// 		s, err := w.chans.AnswerSDP(o, m.Name)
// 		if err != nil {
// 			return err
// 		}
// 		answer := s.SDP()
// 		w.send <- wsMsg{
// 			Type: "answer",
// 			SDP:  &answer,
// 		}
// 		w.rtc = s
// 		return nil
// 	case "play":
// 		if w.rtc != nil {
// 			w.rtc.Close()
// 			w.rtc = nil
// 		}
// 		var p playrtc.PlayRequest
// 		p.Remote = w.conn.RemoteAddr().String()
// 		p.SendCandidate = func(cand webrtc.ICECandidateInit) {
// 			w.send <- wsMsg{
// 				Type:      "candidate",
// 				Candidate: &cand,
// 			}
// 		}
// 		s, err := w.chans.OfferSDP(p, m.Name)
// 		if err != nil {
// 			return err
// 		}
// 		offer := s.SDP()
// 		w.send <- wsMsg{
// 			Type: "offer",
// 			SDP:  &offer,
// 		}
// 		w.rtc = s
// 		return nil
// 	case "answer":
// 		if m.SDP == nil {
// 			return errors.New("missing sdp")
// 		}
// 		if w.rtc == nil {
// 			return errors.New("no session")
// 		}
// 		return w.rtc.SetAnswer(*m.SDP)
// 	case "candidate":
// 		if m.Candidate == nil {
// 			return errors.New("missing candidate")
// 		}
// 		if w.rtc != nil {
// 			w.rtc.Candidate(*m.Candidate)
// 		}
// 		return nil
// 	case "stop":
// 		if w.rtc != nil {
// 			w.rtc.Close()
// 			w.rtc = nil
// 		}
// 		return nil
// 	default:
// 		return errors.New("invalid message type " + m.Type)
// 	}
// }
