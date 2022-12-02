package web

import (
	"encoding/json"
	"errors"
	"net/http"

	"eaglesong.dev/gunk/model"
	"github.com/gorilla/mux"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/hlog"
)

type defsResponse struct {
	Channels []*model.ChannelDef `json:"channels"`
	FTL      string              `json:"ftl"`
}

func (s *Server) viewDefs(rw http.ResponseWriter, req *http.Request) {
	userID := s.checkAuth(rw, req)
	if userID == "" {
		return
	}
	defs, err := model.ListChannelDefs(req.Context(), userID)
	if err != nil {
		hlog.FromRequest(req).Err(err).Msg("failed listing channels")
		http.Error(rw, "", 500)
	}
	for _, def := range defs {
		def.SetURL(s.AdvertiseRTMP)
	}
	ftl, _ := json.MarshalIndent(map[string]interface{}{
		"name":   req.Host,
		"common": true,
		"servers": []map[string]interface{}{
			{
				"name": req.Host,
				"url":  s.AdvertiseLive.Hostname(),
			},
		},
		"recommended": map[string]interface{}{
			"keyint":            2,
			"output":            "ftl_output",
			"max audio bitrate": 160,
			"max video bitrate": 7000,
			"profile":           "main",
			"bframes":           0,
		},
	}, "", "  ")
	res := defsResponse{
		Channels: defs,
		FTL:      string(ftl) + ",",
	}
	writeJSON(rw, res)
}

type defRequest struct {
	Name string `json:"name"`
}

func (s *Server) viewDefsCreate(rw http.ResponseWriter, req *http.Request) {
	userID := s.checkAuth(rw, req)
	if userID == "" {
		return
	}
	var dr defRequest
	if !parseRequest(rw, req, &dr) {
		return
	}
	def, err := model.CreateChannel(req.Context(), userID, dr.Name)
	if err != nil {
		if pge := new(pgconn.PgError); errors.As(err, &pge) && pge.Code == pgerrcode.UniqueViolation {
			http.Error(rw, "channel name already in use", http.StatusConflict)
			return
		}
		hlog.FromRequest(req).Err(err).Str("channel", dr.Name).Msg("failed to create channel")
		http.Error(rw, "", 500)
		return
	}
	def.SetURL(s.AdvertiseRTMP)
	writeJSON(rw, def)
}

type defUpdate struct {
	Announce bool `json:"announce"`
}

func (s *Server) viewDefsUpdate(rw http.ResponseWriter, req *http.Request) {
	userID := s.checkAuth(rw, req)
	if userID == "" {
		return
	}
	var du defUpdate
	if !parseRequest(rw, req, &du) {
		return
	}
	name := mux.Vars(req)["name"]
	if err := model.UpdateChannel(req.Context(), userID, name, du.Announce); err != nil {
		hlog.FromRequest(req).Err(err).Str("channel", name).Msg("failed to update channel")
		http.Error(rw, "", 500)
		return
	}
	writeJSON(rw, nil)
}

func (s *Server) viewDefsDelete(rw http.ResponseWriter, req *http.Request) {
	userID := s.checkAuth(rw, req)
	if userID == "" {
		return
	}
	name := mux.Vars(req)["name"]
	if err := model.DeleteChannel(req.Context(), userID, name); err != nil {
		hlog.FromRequest(req).Err(err).Str("channel", name).Msg("failed to delete channel")
		return
	}
	writeJSON(rw, nil)
}
