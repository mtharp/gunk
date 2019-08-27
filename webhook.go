package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type webhookResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
}

func checkWebhook(ctx context.Context, uri string) (res webhookResponse, err error) {
	err = httpGet(ctx, http.DefaultClient, uri, &res)
	return
}

type webhookMessage struct {
	Content string `json:"content"`
}

func (s *gunkServer) doWebhook(auth channelAuth) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	msg := fmt.Sprintf("<@%s> is now live at %s/watch/%s", auth.UserID, s.baseURL, url.PathEscape(auth.Name))
	blob, _ := json.Marshal(webhookMessage{Content: msg})
	req, err := http.NewRequest("POST", s.webhookURL, bytes.NewReader(blob))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	blob, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %s on webhook: %s", resp.Status, string(blob))
	}
	return nil
}
