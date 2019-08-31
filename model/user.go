package model

import (
	"encoding/json"

	"golang.org/x/oauth2"
)

func SetUser(userID string, token *oauth2.Token, announce bool) error {
	blob, err := json.Marshal(token)
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO users (user_id, refresh_token, announce) VALUES ($1, $2, $3) ON CONFLICT (user_id) DO UPDATE SET refresh_token = EXCLUDED.refresh_token, announce = EXCLUDED.announce", userID, string(blob), announce)
	return err
}
