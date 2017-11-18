package service

import (
	"encoding/json"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
)

// Represents a JSON struct of stats that are updated every second and pushed to the client
type CountUpdate struct {
	Total          string `json:"total"`
	UniqueUsers    string `json:"unique_users"`
	UniqueGuilds   string `json:"unique_guilds"`
	UniqueChannels string `json:"unique_channels"`
}

func (c *CountUpdate) ToJSON() []byte {
	data, _ := json.Marshal(c)
	return data
}

func GetStats() *CountUpdate {
	var (
		total  int64
		users  int64
		guilds int64
		chans  int64
	)

	conn := redisPool.Get()
	defer conn.Close()

	r, err := conn.Do("GET", "airhorn:total")
	if r != nil || err != nil {
		total, err = redis.Int64(r, err)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warning("Failed to get a count update from redis")
		}
	}

	r, err = conn.Do("SCARD", "airhorn:users")
	if r != nil || err != nil {
		users, err = redis.Int64(r, err)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warning("Failed to get a count update from redis")
		}
	}

	r, err = conn.Do("SCARD", "airhorn:guilds")
	if r != nil || err != nil {
		guilds, err = redis.Int64(r, err)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warning("Failed to get a count update from redis")
		}
	}

	r, err = conn.Do("SCARD", "airhorn:channels")
	if r != nil || err != nil {
		chans, err = redis.Int64(r, err)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warning("Failed to get a count update from redis")
		}
	}

	return &CountUpdate{
		Total:          strconv.FormatInt(total, 10),
		UniqueUsers:    strconv.FormatInt(users, 10),
		UniqueGuilds:   strconv.FormatInt(guilds, 10),
		UniqueChannels: strconv.FormatInt(chans, 10),
	}
}
