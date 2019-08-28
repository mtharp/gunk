package model

func SetUser(userID, refreshToken string, announce bool) error {
	_, err := db.Exec("INSERT INTO users (user_id, refresh_token, announce) VALUES ($1, $2, $3) ON CONFLICT (user_id) DO UPDATE SET refresh_token = EXCLUDED.refresh_token, announce = EXCLUDED.announce", userID, refreshToken, announce)
	return err
}
