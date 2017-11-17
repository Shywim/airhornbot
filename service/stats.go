package service

import "encoding/json"

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
