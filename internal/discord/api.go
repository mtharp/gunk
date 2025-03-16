package discord

import (
	"context"
	"net/http"

	"eaglesong.dev/gunk/internal/httpresponse"
)

const (
	discordBase = "https://discord.com/api"
	discordAPI  = discordBase + "/v6"
	discordCDN  = "https://cdn.discordapp.com"
)

func httpGet(ctx context.Context, cli *http.Client, uri string, body any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return err
	}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return httpresponse.NewJSON(resp, body)
}
