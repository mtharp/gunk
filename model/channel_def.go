package model

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/url"

	"github.com/jackc/pgx"
)

type ChannelDef struct {
	Name     string `json:"name"`
	Key      string `json:"key"`
	Announce bool   `json:"announce"`

	RTMPDir  string `json:"rtmp_dir"`
	RTMPBase string `json:"rtmp_base"`
}

func (d *ChannelDef) SetURL(base string) {
	v := url.Values{"key": []string{d.Key}}
	d.RTMPDir = base
	d.RTMPBase = url.PathEscape(d.Name) + "?" + v.Encode()
}

func ListChannelDefs(userID string) (defs []*ChannelDef, err error) {
	rows, err := db.Query("SELECT name, key, announce FROM channel_defs WHERE user_id = $1", userID)
	if err != nil {
		return
	}
	defer rows.Close()
	defs = []*ChannelDef{}
	for rows.Next() {
		def := new(ChannelDef)
		if err = rows.Scan(&def.Name, &def.Key, &def.Announce); err != nil {
			return
		}
		defs = append(defs, def)
	}
	err = rows.Err()
	return
}

func CreateChannel(userID, name string) (def *ChannelDef, err error) {
	b := make([]byte, 24)
	if _, err = io.ReadFull(rand.Reader, b); err != nil {
		return
	}
	key := hex.EncodeToString(b)
	_, err = db.Exec("INSERT INTO channel_defs (user_id, name, key, announce) VALUES ($1, $2, $3, true)", userID, name, key)
	if err != nil {
		return
	}
	return &ChannelDef{Name: name, Key: key, Announce: true}, nil
}

func UpdateChannel(userID, name string, announce bool) error {
	tag, err := db.Exec("UPDATE channel_defs SET announce = $1 WHERE user_id = $2 AND name = $3", announce, userID, name)
	if err != nil {
		return err
	} else if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func DeleteChannel(userID, name string) error {
	_, err := db.Exec("DELETE FROM channel_defs WHERE user_id = $1 AND name = $2", userID, name)
	return err
}
