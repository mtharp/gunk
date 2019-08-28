package model

import (
	"crypto/hmac"
	"crypto/sha512"
	"log"
	"net/url"
	"path"

	"github.com/jackc/pgx"
)

type ChannelAuth struct {
	UserID   string
	Name     string
	Announce bool
}

func findChannel(column, value string) (auth ChannelAuth, key string, err error) {
	row := db.QueryRow("SELECT user_id, channel_defs.name, channel_defs.key, COALESCE(channel_defs.announce AND users.announce, false) FROM channel_defs LEFT JOIN users USING (user_id) WHERE "+column+" = $1", value)
	err = row.Scan(&auth.UserID, &auth.Name, &key, &auth.Announce)
	return
}

func VerifyRTMP(u *url.URL) (auth ChannelAuth, err error) {
	var expectKey string
	auth, expectKey, err = findChannel("name", path.Base(u.Path))
	if err != nil {
		if err == pgx.ErrNoRows {
			err = ErrUserNotFound
		}
		return
	}
	key := u.Query().Get("key")
	if !hmac.Equal([]byte(key), []byte(expectKey)) {
		log.Printf("error: key mismatch for RTMP channel %s", auth.Name)
		err = ErrUserNotFound
		return
	}
	return
}

func VerifyFTL(channelID string, nonce, hmacProvided []byte) (auth ChannelAuth, err error) {
	var expectKey string
	auth, expectKey, err = findChannel("ftl_id", channelID)
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
		log.Printf("error: hmac digest mismatch for FTL channel %s", auth.Name)
		err = ErrUserNotFound
		return
	}
	return
}
