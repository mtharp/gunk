package web

import (
	"testing"

	"eaglesong.dev/gunk/ui/src/router"
	"github.com/stretchr/testify/assert"
)

func TestRoutes(t *testing.T) {
	assert.ElementsMatch(t, []string{
		"/",
		"/mychannels",
		"/watch/{channel}",
	}, router.IndexRoutes())
}
