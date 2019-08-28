package model

import (
	"errors"

	"github.com/jackc/pgx"
)

var db *pgx.ConnPool

func Connect() error {
	conf, err := pgx.ParseEnvLibpq()
	if err != nil {
		return err
	}
	db, err = pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:     conf,
		MaxConnections: 10,
	})
	return err
}

var ErrUserNotFound = errors.New("user not found or wrong key")
