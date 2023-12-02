package internal

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

func RandomID(nbytes int) string {
	d := make([]byte, nbytes)
	if _, err := io.ReadFull(rand.Reader, d); err != nil {
		panic("random source unavailable")
	}
	return hex.EncodeToString(d)
}
