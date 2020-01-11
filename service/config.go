package service

import (
	"fmt"

	"github.com/spf13/viper"
)

// Cfg represents the app configuration
type Cfg struct {
	DBDriver            string
	DBSSL               bool
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	RedisHost           string
	DiscordToken        string
	DiscordClientID     string
	DiscordClientSecret string
	DiscordRedirectURI  string
	DataPath            string
	PluginPath          string
	DiscordOwnerID      string
	WebURL              string
}

var config Cfg

// LoadConfig read configuration from disk
func LoadConfig() (Cfg, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath("config")
	viper.AddConfigPath("/etc/airhornbot")

	err := viper.ReadInConfig()
	if err != nil {
		return Cfg{}, err
	}

	cfg := Cfg{}
	cfg.DBDriver = viper.GetString("database.driver")
	cfg.DBSSL = viper.GetBool("database.ssl")
	cfg.DBHost = viper.GetString("database.host")
	cfg.DBPort = viper.GetString("database.port")
	cfg.DBUser = viper.GetString("database.user")
	cfg.DBPassword = viper.GetString("database.password")
	cfg.DBName = viper.GetString("database.name")
	cfg.RedisHost = viper.GetString("redis.host")
	cfg.DiscordToken = viper.GetString("discord.token")
	cfg.DiscordClientID = viper.GetString("discord.client_id")
	cfg.DiscordClientSecret = viper.GetString("discord.client_secret")
	cfg.DiscordRedirectURI = viper.GetString("discord.redirect_uri")
	cfg.DataPath = viper.GetString("data.data_path")
	cfg.PluginPath = viper.GetString("data.plugins_path")
	cfg.DiscordOwnerID = viper.GetString("discord.owner_id")
	cfg.WebURL = viper.GetString("web.url")

	if cfg.DBDriver == "mysql" {
		cfg.DBHost = fmt.Sprintf("tcp(%s:%s)", cfg.DBHost, cfg.DBPort)
	} else if cfg.DBDriver == "postgres" {
		cfg.DBHost = fmt.Sprintf("%s:%s", cfg.DBHost, cfg.DBPort)
	}

	config = cfg

	go initDb()

	return cfg, nil
}
