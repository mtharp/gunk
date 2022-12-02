package model

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

func Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var err error
	db, err = pgxpool.New(ctx, "")
	return err
}

var ErrUserNotFound = errors.New("user not found or wrong key")
