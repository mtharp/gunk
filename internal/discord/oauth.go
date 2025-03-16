package discord

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

var Endpoint = oauth2.Endpoint{
	AuthURL:   discordBase + "/oauth2/authorize",
	TokenURL:  discordBase + "/oauth2/token",
	AuthStyle: oauth2.AuthStyleInHeader,
}

type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`

	GuildIDs []string `json:"-"`
}

func CurrentUser(ctx context.Context, tsrc oauth2.TokenSource) (User, error) {
	cli := oauth2.NewClient(ctx, tsrc)
	var user User
	if err := httpGet(ctx, cli, discordAPI+"/users/@me", &user); err != nil {
		return User{}, fmt.Errorf("getting current user's info: %w", err)
	}
	var guildList []struct {
		ID string `json:"id"`
	}
	if err := httpGet(ctx, cli, discordAPI+"/users/@me/guilds", &guildList); err != nil {
		return User{}, fmt.Errorf("getting current user's guild list: %w", err)
	}
	guilds := make([]string, len(guildList))
	for i, guild := range guildList {
		guilds[i] = guild.ID
	}
	return user, nil
}
