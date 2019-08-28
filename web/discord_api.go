package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"eaglesong.dev/gunk/model"
	"golang.org/x/oauth2"
)

type discordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
}

func httpGet(ctx context.Context, cli *http.Client, uri string, body interface{}) error {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return err
	}
	resp, err := cli.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	blob, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	} else if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %s on %s %s:\n%s", resp.Status, resp.Request.Method, resp.Request.URL, string(blob))
	}
	return json.Unmarshal(blob, body)
}

func (s *Server) lookupUser(ctx context.Context, token *oauth2.Token) (user discordUser, err error) {
	cli := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	if err = httpGet(ctx, cli, "https://discordapp.com/api/users/@me", &user); err != nil {
		return
	}
	var guildList []struct {
		ID string `json:"id"`
	}
	if err = httpGet(ctx, cli, "https://discordapp.com/api/users/@me/guilds", &guildList); err != nil {
		return
	}
	var announce bool
	for _, guild := range guildList {
		if guild.ID == s.checkGuild {
			announce = true
		}
	}
	err = model.SetUser(user.ID, token.RefreshToken, announce)
	return
}

type webhookResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
}

func (s *Server) SetWebhook(u string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var hook webhookResponse
	if err := httpGet(ctx, http.DefaultClient, u, &hook); err != nil {
		return err
	}
	s.webhookURL = u
	s.checkGuild = hook.GuildID
	return nil
}

type webhookMessage struct {
	Content string `json:"content"`
}

func (s *Server) doWebhook(auth model.ChannelAuth) error {
	if s.webhookURL == "" || !auth.Announce {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	msg := fmt.Sprintf("<@%s> is now live at %s/watch/%s", auth.UserID, s.BaseURL, url.PathEscape(auth.Name))
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
