package model

import "context"

func GetThumb(ctx context.Context, channelName string) (d []byte, err error) {
	row := db.QueryRow(ctx, "SELECT thumb FROM thumbs WHERE name = $1", channelName)
	err = row.Scan(&d)
	return
}

func PutThumb(ctx context.Context, channelName string, d []byte) error {
	_, err := db.Exec(ctx, "INSERT INTO thumbs (name, thumb) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET thumb = EXCLUDED.thumb, updated = now()", channelName, d)
	return err
}
