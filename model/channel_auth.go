package model

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

type ChannelAuth struct {
	UserID   string
	Name     string
	Announce bool
	Token    *oauth2.Token
}

func findChannel(ctx context.Context, column, value string) (auth ChannelAuth, key string, err error) {
	row := db.QueryRow(ctx, "SELECT user_id, channel_defs.name, channel_defs.key, users.refresh_token, COALESCE(channel_defs.announce AND users.announce, false) FROM channel_defs LEFT JOIN users USING (user_id) WHERE "+column+" = $1", value)
	var blob *string
	err = row.Scan(&auth.UserID, &auth.Name, &key, &blob, &auth.Announce)
	if err != nil || blob == nil || *blob == "" {
		return
	}
	err = json.Unmarshal([]byte(*blob), &auth.Token)
	return
}

func VerifyPassword(ctx context.Context, channel, password string) (auth ChannelAuth, err error) {
	var expectKey string
	auth, expectKey, err = findChannel(ctx, "name", channel)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = ErrUserNotFound
		}
		return
	}
	if !hmac.Equal([]byte(password), []byte(expectKey)) {
		log.Printf("error: key mismatch for RTMP channel %s", auth.Name)
		err = ErrUserNotFound
		return
	}
	return
}

func VerifyFTL(ctx context.Context, channelID string, nonce, hmacProvided []byte) (auth ChannelAuth, err error) {
	var expectKey string
	auth, expectKey, err = findChannel(ctx, "ftl_id", channelID)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = ErrUserNotFound
		}
		return
	}
	hm := hmac.New(sha512.New, []byte(expectKey))
	hm.Write(nonce)
	expected := hm.Sum(nil)
	if !hmac.Equal(expected, hmacProvided) {
		log.Error().Str("channel", auth.Name).Msg("hmac digest mismatch in FTL auth")
		err = ErrUserNotFound
		return
	}
	return
}
