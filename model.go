// Copyright © Michael Tharp <gxti@partiallystapled.com>
//
// This file is distributed under the terms of the MIT License.
// See the LICENSE file at the top of this tree or http://opensource.org/licenses/MIT

package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx"
)

var db *pgx.ConnPool

func connectDB() error {
	conf, err := pgx.ParseEnvLibpq()
	if err != nil {
		return err
	}
	db, err = pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:     conf,
		MaxConnections: 10,
	})
	return err
}

type channelDef struct {
	Name string `json:"name"`
	Key  string `json:"key"`

	RTMPDir  string `json:"rtmp_dir"`
	RTMPBase string `json:"rtmp_base"`
}

func (d *channelDef) setURL(base string) {
	v := url.Values{"key": []string{d.Key}}
	d.RTMPDir = base
	d.RTMPBase = url.PathEscape(d.Name) + "?" + v.Encode()
}

func getChannelDefs(userID string) (defs []*channelDef, err error) {
	rows, err := db.Query("SELECT name, key FROM channel_defs WHERE user_id = $1", userID)
	if err != nil {
		return
	}
	defer rows.Close()
	defs = []*channelDef{}
	for rows.Next() {
		def := new(channelDef)
		if err = rows.Scan(&def.Name, &def.Key); err != nil {
			return
		}
		defs = append(defs, def)
	}
	err = rows.Err()
	return
}

func createChannel(userID, name string) (def *channelDef, err error) {
	b := make([]byte, 24)
	if _, err = io.ReadFull(rand.Reader, b); err != nil {
		return
	}
	key := hex.EncodeToString(b)
	_, err = db.Exec("INSERT INTO channel_defs (user_id, name, key) VALUES ($1, $2, $3)", userID, name, key)
	if err != nil {
		return
	}
	return &channelDef{Name: name, Key: key}, nil
}

func deleteChannel(userID, name string) error {
	_, err := db.Exec("DELETE FROM channel_defs WHERE user_id = $1 AND name = $2", userID, name)
	return err
}

func verifyChannel(u *url.URL) (userID string) {
	name := path.Base(u.Path)
	key := u.Query().Get("key")
	row := db.QueryRow("SELECT user_id FROM channel_defs WHERE name = $1 AND key = $2", name, key)
	if err := row.Scan(&userID); err == pgx.ErrNoRows {
		return ""
	} else if err != nil {
		log.Printf("error: querying database: %s", err)
		return ""
	}
	return userID
}

func getThumb(channelName string) (d []byte, err error) {
	row := db.QueryRow("SELECT thumb FROM thumbs WHERE name = $1", channelName)
	err = row.Scan(&d)
	return
}

func putThumb(channelName string, d []byte) error {
	_, err := db.Exec("INSERT INTO thumbs (name, thumb) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET thumb = EXCLUDED.thumb, updated = now()", channelName, d)
	return err
}

func (s *gunkServer) checkAuth(rw http.ResponseWriter, req *http.Request) string {
	var info loginInfo
	err := s.unseal(req, s.loginCookie, &info)
	if err == nil {
		return info.ID
	}
	log.Printf("error: authentication failed for %s to %s", req.RemoteAddr, req.URL)
	http.Error(rw, "not authorized", 401)
	return ""
}

func (s *gunkServer) viewDefs(rw http.ResponseWriter, req *http.Request) {
	userID := s.checkAuth(rw, req)
	if userID == "" {
		return
	}
	defs, err := getChannelDefs(userID)
	if err != nil {
		log.Println("error:", err)
		http.Error(rw, "", 500)
	}
	for _, def := range defs {
		def.setURL(s.rtmpBase)
	}
	blob, _ := json.Marshal(defs)
	rw.Header().Set("Content-Type", "application/json")
	rw.Write(blob)
}

func parseRequest(rw http.ResponseWriter, req *http.Request, d interface{}) bool {
	blob, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("error: reading %s request: %s", req.RemoteAddr, err)
		http.Error(rw, "", 500)
		return false
	}
	if err := json.Unmarshal(blob, d); err != nil {
		log.Printf("error: reading %s request: %s", req.RemoteAddr, err)
		http.Error(rw, "invalid JSON in request", 400)
		return false
	}
	return true
}

type defRequest struct {
	Name string `json:"name"`
}

func (s *gunkServer) viewDefsCreate(rw http.ResponseWriter, req *http.Request) {
	userID := s.checkAuth(rw, req)
	if userID == "" {
		return
	}
	var dr defRequest
	if !parseRequest(rw, req, &dr) {
		return
	}
	def, err := createChannel(userID, dr.Name)
	if err != nil {
		if pge, ok := err.(pgx.PgError); ok && pge.Code == "23505" {
			http.Error(rw, "channel name already in use", http.StatusConflict)
			return
		}
		log.Printf("error: creating channel %q for %s: %s", dr.Name, req.RemoteAddr, err)
		http.Error(rw, "", 500)
		return
	}
	def.setURL(s.rtmpBase)
	blob, _ := json.Marshal(def)
	rw.Header().Set("Content-Type", "application/json")
	rw.Write(blob)
}

func (s *gunkServer) viewDefsDelete(rw http.ResponseWriter, req *http.Request) {
	userID := s.checkAuth(rw, req)
	if userID == "" {
		return
	}
	name := mux.Vars(req)["name"]
	if err := deleteChannel(userID, name); err != nil {
		log.Printf("error: deleting channel %q for %s: %s", name, req.RemoteAddr, err)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.Write([]byte("{}"))
}

type channelInfo struct {
	Name string `json:"name"`
	Live bool   `json:"live"`
	Last int64  `json:"last"`
}

func listChannels() (ret []channelInfo, err error) {
	rows, err := db.Query("SELECT name, updated FROM thumbs ORDER BY greatest(now() - updated, '1 minute'::interval) ASC, 1 ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var info channelInfo
		var last time.Time
		if err := rows.Scan(&info.Name, &last); err != nil {
			return nil, err
		}
		info.Last = last.UnixNano() / 1000000
		ret = append(ret, info)
	}
	err = rows.Err()
	return
}
