package model

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"math/big"
	"net/url"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ChannelDef struct {
	Name     string `json:"name"`
	Key      string `json:"key"`
	Announce bool   `json:"announce"`
	FTLKey   string `json:"ftl_key"`

	RTMPDir  string `json:"rtmp_dir"`
	RTMPBase string `json:"rtmp_base"`
}

func (d *ChannelDef) SetURL(base string) {
	v := url.Values{"key": []string{d.Key}}
	d.RTMPDir = base
	d.RTMPBase = url.PathEscape(d.Name) + "?" + v.Encode()
}

func ListChannelDefs(ctx context.Context, userID string) (defs []*ChannelDef, err error) {
	rows, err := db.Query(ctx, "SELECT name, key, announce, ftl_id FROM channel_defs WHERE user_id = $1", userID)
	if err != nil {
		return
	}
	defer rows.Close()
	defs = []*ChannelDef{}
	for rows.Next() {
		def := new(ChannelDef)
		var ftlID pgtype.Text
		if err = rows.Scan(&def.Name, &def.Key, &def.Announce, &ftlID); err != nil {
			return
		}
		def.FTLKey = ftlID.String + "," + def.Key
		defs = append(defs, def)
	}
	err = rows.Err()
	return
}

func CreateChannel(ctx context.Context, userID, name string) (def *ChannelDef, err error) {
	b := make([]byte, 24)
	if _, err = io.ReadFull(rand.Reader, b); err != nil {
		return
	}
	key := hex.EncodeToString(b)
	ftlID, _ := rand.Int(rand.Reader, new(big.Int).SetInt64(1<<31))
	_, err = db.Exec(ctx, "INSERT INTO channel_defs (user_id, name, key, announce, ftl_id) VALUES ($1, $2, $3, true, $4)",
		userID, name, key, ftlID.String())
	if err != nil {
		return
	}
	return &ChannelDef{
		Name:     name,
		Key:      key,
		Announce: true,
		FTLKey:   ftlID.String() + "," + key,
	}, nil
}

func UpdateChannel(ctx context.Context, userID, name string, announce bool) error {
	tag, err := db.Exec(ctx, "UPDATE channel_defs SET announce = $1 WHERE user_id = $2 AND name = $3", announce, userID, name)
	if err != nil {
		return err
	} else if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func DeleteChannel(ctx context.Context, userID, name string) error {
	_, err := db.Exec(ctx, "DELETE FROM channel_defs WHERE user_id = $1 AND name = $2", userID, name)
	return err
}
