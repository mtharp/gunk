package server

import (
	"net/http"

	"eaglesong.dev/gunk/internal/model"
)

type defsResponse struct {
	Channels []*model.ChannelDef `json:"channels"`
}

func (s *Server) setDef(def *model.ChannelDef) {
	def.SetURL(s.AdvertiseRTMP, s.BaseURL)

}

func (s *Server) viewMyChannels(rw http.ResponseWriter, req *http.Request, userID string) error {
	defs, err := s.Model.ListUserChannels(req.Context(), userID)
	if err != nil {
		return err
	}
	for _, def := range defs {
		s.setDef(def)
	}
	writeJSON(rw, defsResponse{Channels: defs})
	return nil
}

type defRequest struct {
	Name string `json:"name"`
}

func (s *Server) viewChannelCreate(rw http.ResponseWriter, req *http.Request, userID string) error {
	var dr defRequest
	if !readJSON(rw, req, &dr) {
		return nil
	}
	def, err := s.Model.CreateChannelDef(req.Context(), userID, dr.Name)
	if err != nil {
		return err
	}
	s.setDef(def)
	writeJSON(rw, def)
	return nil
}

type defUpdate struct {
	Announce bool `json:"announce"`
}

func (s *Server) viewChannelUpdate(rw http.ResponseWriter, req *http.Request, userID string) error {
	var du defUpdate
	if !readJSON(rw, req, &du) {
		return nil
	}
	name := req.PathValue("channel")
	def, err := s.Model.UpdateChannelDef(req.Context(), userID, name, du.Announce)
	if err != nil {
		return err
	}
	s.setDef(def)
	writeJSON(rw, def)
	return nil
}

func (s *Server) viewChannelDelete(rw http.ResponseWriter, req *http.Request, userID string) error {
	name := req.PathValue("channel")
	if err := s.Model.DeleteChannelDef(req.Context(), userID, name); err != nil {
		return err
	}
	rw.WriteHeader(http.StatusNoContent)
	return nil
}
