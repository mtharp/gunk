package model

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// type ChannelAuth struct {
// 	UserID   string
// 	Name     string
// 	Announce bool
// }

// func findChannel(ctx context.Context, column, value string) (auth ChannelAuth, key string, err error) {
// 	row := db.QueryRow(ctx, "SELECT user_id, channel_defs.name, channel_defs.key, COALESCE(channel_defs.announce AND users.announce, false) FROM channel_defs LEFT JOIN users USING (user_id) WHERE "+column+" = $1", value)
// 	var blob *string
// 	err = row.Scan(&auth.UserID, &auth.Name, &key, &auth.Announce)
// 	if err != nil || blob == nil || *blob == "" {
// 		return
// 	}
// 	return
// }

// func (d *db) GetChannel(ctx context.Context, name string) (ChannelAuth, error) {
// 	auth, _, err := findChannel(ctx, "name", name)
// 	return auth, err
// }

func (d *db) VerifyPassword(ctx context.Context, channel, password string) (ChannelDef, error) {
	row := d.pool.QueryRow(ctx, "SELECT user_id, channel_defs.name, channel_defs.key, "+
		"channel_defs.announce, users.refresh_token FROM channel_defs "+
		"LEFT JOIN users USING (user_id) WHERE channel_defs.name = $1", channel)
	var def ChannelDef
	var expectKey string
	var tokenBlob []byte
	if err := row.Scan(&def.UserID, &def.Name, &expectKey, &def.Announce, &tokenBlob); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = ErrUserNotFound
		}
		return ChannelDef{}, err
	}
	if !hmac.Equal([]byte(password), []byte(expectKey)) {
		return ChannelDef{}, ErrUserNotFound
	}
	if err := json.Unmarshal(tokenBlob, &def.RefreshToken); err != nil {
		return ChannelDef{}, fmt.Errorf("unmarshalling refresh_token: %w", err)
	}
	return def, nil
}
