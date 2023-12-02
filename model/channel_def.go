package model

import (
	"context"
	"net/url"

	"eaglesong.dev/gunk/internal"
	"github.com/jackc/pgx/v5"
)

type ChannelDef struct {
	Name     string `json:"name"`
	Key      string `json:"key"`
	Announce bool   `json:"announce"`

	RTMPDir  string `json:"rtmp_dir"`
	RTMPBase string `json:"rtmp_base"`

	RISTUrl string `json:"rist_url"`
}

func (d *ChannelDef) SetURL(base string, rist *url.URL) {
	v := url.Values{"key": []string{d.Key}}
	d.RTMPDir = base
	d.RTMPBase = url.PathEscape(d.Name) + "?" + v.Encode()
	if rist != nil {
		u2 := new(url.URL)
		*u2 = *rist
		q := rist.Query()
		q.Set("cname", d.Name)
		u2.RawQuery = q.Encode()
		d.RISTUrl = u2.String()
	}
}

func ListChannelDefs(ctx context.Context, userID string) (defs []*ChannelDef, err error) {
	rows, err := db.Query(ctx, "SELECT name, key, announce FROM channel_defs WHERE user_id = $1", userID)
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

func CreateChannel(ctx context.Context, userID, name string) (def *ChannelDef, err error) {
	key := internal.RandomID(24)
	_, err = db.Exec(ctx, "INSERT INTO channel_defs (user_id, name, key, announce) VALUES ($1, $2, $3, true)",
		userID, name, key)
	if err != nil {
		return
	}
	return &ChannelDef{
		Name:     name,
		Key:      key,
		Announce: true,
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
