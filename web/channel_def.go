package web

import (
	"log"
	"net/http"

	"eaglesong.dev/gunk/model"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx"
)

func (s *Server) viewDefs(rw http.ResponseWriter, req *http.Request) {
	userID := s.checkAuth(rw, req)
	if userID == "" {
		return
	}
	defs, err := model.ListChannelDefs(userID)
	if err != nil {
		log.Println("error:", err)
		http.Error(rw, "", 500)
	}
	for _, def := range defs {
		def.SetURL(s.AdvertiseRTMP)
	}
	writeJSON(rw, defs)
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
	def, err := model.CreateChannel(userID, dr.Name)
	if err != nil {
		if pge, ok := err.(pgx.PgError); ok && pge.Code == "23505" {
			http.Error(rw, "channel name already in use", http.StatusConflict)
			return
		}
		log.Printf("error: creating channel %q for %s: %s", dr.Name, req.RemoteAddr, err)
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
	if err := model.UpdateChannel(userID, name, du.Announce); err != nil {
		log.Printf("error: updating channel %q for %s: %s", name, req.RemoteAddr, err)
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
	if err := model.DeleteChannel(userID, name); err != nil {
		log.Printf("error: deleting channel %q for %s: %s", name, req.RemoteAddr, err)
		return
	}
	writeJSON(rw, nil)
}
