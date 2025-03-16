package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"eaglesong.dev/gunk/internal/httpresponse"
)

type Webhook struct {
	URL string `json:"-"`

	Name      string `json:"name"`
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
}

func NewWebhook(ctx context.Context, url string) (*Webhook, error) {
	hook := new(Webhook)
	if err := httpGet(ctx, http.DefaultClient, url, hook); err != nil {
		return nil, err
	}
	hook.URL = url
	return hook, nil
}

func (w *Webhook) Invoke(ctx context.Context, content string) error {
	body := map[string]any{
		"content": content,
		// @ users but don't ping them
		"allowed_mentions": map[string]any{
			"parse": []struct{}{},
		},
	}
	blob, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(blob))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode > http.StatusNoContent {
		return httpresponse.NewError(resp)
	}
	return nil
}
