package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/zerolog/hlog"
)

func (s *Server) viewPlaylist(rw http.ResponseWriter, req *http.Request) {
	channel := req.PathValue("channel")
	channel = strings.TrimSuffix(channel, ".m3u8")

	livePaths, err := s.getLivePaths(req.Context())
	if err != nil {
		hlog.FromRequest(req).Err(err).Msg("failed to list live paths")
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}
	status := livePaths[channel]
	if status.Name == "" {
		http.NotFound(rw, req)
		return
	}
	rw.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	fmt.Fprintf(rw, "rtsp://127.0.0.1:8554/%s\n", url.PathEscape(status.Name))
}
