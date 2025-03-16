package model

import (
	"context"
	"database/sql"
	"time"
)

func (d *db) GetThumb(ctx context.Context, channelName string) (blob []byte, last time.Time, err error) {
	row := d.pool.QueryRow(ctx, "SELECT thumb, updated FROM thumbs WHERE name = $1", channelName)
	err = row.Scan(&blob, &last)
	return
}

func (d *db) PutThumb(ctx context.Context, channelName string, blob []byte) (bool, time.Time, error) {
	row := d.pool.QueryRow(ctx,
		`WITH old_thumb AS (
			SELECT name, updated FROM thumbs WHERE name = $1
		), new_thumb AS (
			INSERT INTO thumbs (name, thumb) VALUES ($1, $2)
			ON CONFLICT (name) DO UPDATE SET thumb = EXCLUDED.thumb, updated = now()
		)
		SELECT announce, old_thumb.updated FROM channel_defs
		LEFT JOIN old_thumb USING (name)
		WHERE name = $1`,
		channelName, blob)
	var announce bool
	var old sql.Null[time.Time]
	err := row.Scan(&announce, &old)
	return announce, old.V, err
}
