package model

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type db struct {
	pool *pgxpool.Pool
}

func Connect() (*db, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, "")
	if err != nil {
		return nil, err
	}
	return &db{
		pool: pool,
	}, nil
}

var ErrUserNotFound = errors.New("user not found or wrong key")
