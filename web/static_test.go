package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodings(t *testing.T) {
	runs := []struct {
		Name, Header string
		Brotli, Gzip bool
	}{
		{"Empty", "", false, false},
		{"Gzip", "gzip, identity", false, true},
		{"All", "gzip, deflate, br", true, true},
		{"Q", "gzip;q=1.0, br, *;q=0.5", true, true},
	}
	for _, r := range runs {
		r := r
		t.Run(r.Name, func(t *testing.T) {
			b, g := checkEncodings(r.Header)
			assert.Equal(t, r.Brotli, b)
			assert.Equal(t, r.Gzip, g)
		})
	}
}
