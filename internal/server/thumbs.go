package server

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/hlog"
)

const announceCooldown = 5 * time.Minute

func (s *Server) callbackThumbnail(rw http.ResponseWriter, req *http.Request) {
	channelName := req.PathValue("channel")
	body := readRequest(rw, req, 250e6)
	if len(body) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	announce, previous, err := s.Model.PutThumb(ctx, channelName, body)
	if err != nil {
		hlog.FromRequest(req).Err(err).Msg("failed to update thumbnail")
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusNoContent)
}

func (s *Server) viewThumbnail(rw http.ResponseWriter, req *http.Request) {
	channelName := req.PathValue("channel")
	jpegBlob, mtime, err := s.Model.GetThumb(req.Context(), channelName)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(rw, req)
		return
	} else if err != nil {
		hlog.FromRequest(req).Err(err).Msg("failed to get thumbnail")
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Cache-Control", "max-age=86400, public, immutable")
	if req.Header.Get("Origin") != "" {
		rw.Header().Set("Access-Control-Allow-Origin", "*")
	}
	http.ServeContent(rw, req, ".jpeg", mtime, bytes.NewReader(jpegBlob))
}
