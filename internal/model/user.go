package model

import (
	"context"
	"encoding/json"

	"golang.org/x/oauth2"
)

func (d *db) CreateOrUpdateUser(ctx context.Context, userID string, refreshToken *oauth2.Token) error {
	tokenBlob, err := json.Marshal(refreshToken)
	if err != nil {
		return err
	}
	_, err = d.pool.Exec(ctx, "INSERT INTO users (user_id, refresh_token) VALUES ($1, $2) "+
		"ON CONFLICT (user_id) DO UPDATE SET refresh_token = EXCLUDED.refresh_token",
		userID, tokenBlob)
	return err
}
