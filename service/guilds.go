package service

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/garyburd/redigo/redis"
)

const (
	permAdministrator = 8
)

// Guild represents a discord server
type Guild struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Icon   string   `json:"icon"`
	Sounds []*Sound `json:"sounds"`
}

// UserGuilds represents a user's guilds
type UserGuilds struct {
	AirhornGuilds []*Guild
	BoringGuilds  []*Guild
}

// AddGuild register a new guild to use Airhorn
func AddGuild(gID string) error {
	// Store the guild id in redis
	conn := redisPool.Get()
	_, err := conn.Do("SADD", "airhorn:guilds:list", gID)
	return err
}

// GetGuildWithSounds retrieves a guild from Discord and its sounds
func GetGuildWithSounds(session *discordgo.Session, gID string) (Guild, error) {
	guilds, err := session.UserGuilds(100, "", "")
	if err != nil {
		return Guild{}, err
	}

	for _, g := range guilds {
		if g.ID != gID {
			continue
		}
		guild := Guild{
			ID:   g.ID,
			Name: g.Name,
			Icon: fmt.Sprintf("https://cdn.discordapp.com/icons/%v/%v.png",
				g.ID, g.Icon),
		}

		sounds, err := GetSoundsByGuild(g.ID)
		if err != nil {
			return Guild{}, err
		}
		guild.Sounds = sounds

		return guild, nil
	}
	return Guild{}, errors.New("no guild found")
}

// GetGuildsWithSounds retrieves a guild from Discord and its sounds
func GetGuildsWithSounds(session *discordgo.Session) (*UserGuilds, error) {
	guilds, err := session.UserGuilds(100, "", "")
	if err != nil {
		return nil, err
	}

	var airhornGuilds []*Guild
	var boringGuilds []*Guild
	for _, g := range guilds {
		guild := &Guild{
			ID:   g.ID,
			Name: g.Name,
			Icon: fmt.Sprintf("https://cdn.discordapp.com/icons/%v/%v.png",
				g.ID, g.Icon),
		}

		if g.Permissions&permAdministrator != 0 {
			hasAirhorn, err := GuildHasAirhorn(g.ID)
			if err != nil {
				// TODO: error
				continue
			}

			if hasAirhorn {
				boringGuilds = append(boringGuilds, guild)
				continue
			}

			sounds, err := GetSoundsByGuild(g.ID)
			guild.Sounds = sounds

			airhornGuilds = append(airhornGuilds, guild)
		}
	}

	return &UserGuilds{
		AirhornGuilds: airhornGuilds,
		BoringGuilds:  boringGuilds,
	}, nil
}

// GuildHasAirhorn checks if a guild has already been used with airhorn
func GuildHasAirhorn(gID string) (bool, error) {
	conn := redisPool.Get()
	defer conn.Close()

	r, err := conn.Do("SISMEMBER", "airhorn:guilds:list", gID)
	hasAirhorn, err := redis.Int64(r, err)
	if err != nil {
		return false, err
	}

	return hasAirhorn == 0, nil
}
