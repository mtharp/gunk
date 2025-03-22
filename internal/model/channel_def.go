package model

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"eaglesong.dev/gunk/internal"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kr/pretty"
	"golang.org/x/oauth2"
)

type ChannelDef struct {
	UserID       string        `json:"-"`
	RefreshToken *oauth2.Token `json:"-"`

	Name     string `json:"name"`
	Key      string `json:"key"`
	Announce bool   `json:"announce"`

	RTMPDir  string `json:"rtmp_dir"`
	RTMPBase string `json:"rtmp_base"`

	WHIPUrl string `json:"whip_url"`
}

func (d *ChannelDef) SetURL(rtmpBase, webBase string) {
	v := url.Values{"key": []string{d.Key}}
	d.RTMPDir = rtmpBase
	d.RTMPBase = url.PathEscape(d.Name) + "?" + v.Encode()
	d.WHIPUrl = webBase + "/whip/" + url.PathEscape(d.Name)
}

func (d *db) ListUserChannels(ctx context.Context, userID string) ([]*ChannelDef, error) {
	rows, err := d.pool.Query(ctx, "SELECT name, key, announce FROM channel_defs WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var defs []*ChannelDef
	for rows.Next() {
		def := &ChannelDef{UserID: userID}
		if err := rows.Scan(&def.Name, &def.Key, &def.Announce); err != nil {
			return nil, err
		}
		defs = append(defs, def)
	}
	return defs, rows.Err()
}

func toChannel(row pgx.Row, name string) (*ChannelDef, error) {
	def := new(ChannelDef)
	err := row.Scan(&def.UserID, &def.Name, &def.Key, &def.Announce)
	var pge *pgconn.PgError
	if !errors.As(err, &pge) {
		return def, err
	}
	pretty.Println(pge)
	switch pge.Code {
	case pgerrcode.UniqueViolation:
		// {Detail:"Key (name)=(asdf) already exists.", ConstraintName:"channel_defs_pkey"}
		err = DuplicateChannelError{Name: name}
	}
	return def, err
}

var validNameRe = regexp.MustCompile(`^[A-Za-z0-9 .,_-]+$`)

func (d *db) CreateChannelDef(ctx context.Context, userID, name string) (*ChannelDef, error) {
	if !validNameRe.MatchString(name) {
		return nil, errors.New("invalid channel name")
	}
	key := internal.RandomID(24)
	row := d.pool.QueryRow(ctx,
		`INSERT INTO channel_defs (user_id, name, key)
		VALUES ($1, $2, $3)
		RETURNING user_id, name, key, announce`,
		userID, name, key)
	return toChannel(row, name)
}

func (d *db) UpdateChannelDef(ctx context.Context, userID, name string, announce bool) (*ChannelDef, error) {
	row := d.pool.QueryRow(ctx,
		`UPDATE channel_defs SET announce = $1
		WHERE user_id = $2 AND name = $3
		RETURNING user_id, name, key, announce`,
		announce, userID, name)
	return toChannel(row, name)
}

func (d *db) DeleteChannelDef(ctx context.Context, userID, name string) error {
	_, err := d.pool.Exec(ctx, "DELETE FROM channel_defs WHERE user_id = $1 AND name = $2", userID, name)
	return err
}

type DuplicateChannelError struct {
	Name string
}

func (e DuplicateChannelError) Error() string {
	return fmt.Sprintf("a channel named %q already exists", e.Name)
}
