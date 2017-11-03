package common

import (
	"database/sql"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	// mysql driver, used via database/sql
	_ "github.com/go-sql-driver/mysql"
	// postgresql driver, used via database/sql
	_ "github.com/lib/pq"
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
	DataPath            string
	DiscordOwnerID      string
}

var config *Cfg

// LoadConfig read configuration from disk
func LoadConfig() *Cfg {
	viper.SetConfigName("config")
	viper.AddConfigPath("config")
	viper.AddConfigPath("/etc/airhornbot")

	err := viper.ReadInConfig()
	if err != nil {
		// TODO: log error
	}

	cfg := &Cfg{}
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
	cfg.DataPath = viper.GetString("data.data_path")
	cfg.DiscordOwnerID = viper.GetString("discord.owner_id")

	if cfg.DBDriver == "mysql" {
		cfg.DBHost = fmt.Sprintf("tcp(%s:%s)", cfg.DBHost, cfg.DBPort)
	} else if cfg.DBDriver == "postgres" {
		cfg.DBHost = fmt.Sprintf("%s:%s", cfg.DBHost, cfg.DBPort)
	}

	config = cfg

	go initDb()

	return cfg
}

// Sound represents a sound clip
type Sound struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// Link to a gif url
	Gif string `json:"gif"`

	// Weight adjust how likely it is this song will play, higher = more likely
	Weight int `json:"weight"`

	// Command to type in channel
	Commands []string `json:"commands"`

	FilePath string
}

func initDb() {
	db, err := getDB()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Couldn't connect to DB")
		return
	}

	primaryKeyType := "INTEGER PRIMARY KEY"
	if config.DBDriver == "postgres" {
		primaryKeyType = "SERIAL PRIMARY KEY"
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS sound (" +
		"id " + primaryKeyType + "," +
		"guildId VARCHAR(255)," +
		"name VARCHAR(255)," +
		"gif VARCHAR(255)," +
		"weight INTEGER," +
		"filepath VARCHAR(255)" +
		")")
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS command (" +
		"id " + primaryKeyType + "," +
		"command VARCHAR(255)," +
		"guildId VARCHAR(255)," +
		"soundId INTEGER," +
		"FOREIGN KEY(soundId) REFERENCES sound(id) ON DELETE CASCADE" +
		")")

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Error creating tables")
	}
}

func buildSound(row *sql.Row) (*Sound, error) {
	var sound Sound
	var gif sql.NullString // gif can be null
	if err := row.Scan(&sound.ID, &sound.Name, &gif, &sound.Weight, &sound.FilePath); err != nil {
		return nil, err
	}

	sound.Gif = gif.String
	return &sound, nil
}

func buildSounds(db *sql.DB, rows *sql.Rows) ([]*Sound, error) {
	var sounds []*Sound
	for rows.Next() {
		var sound Sound
		if err := rows.Scan(&sound.ID, &sound.Name, &sound.Gif, &sound.Weight, &sound.FilePath); err != nil {
			return nil, err
		}

		commandsRows, err := db.Query("SELECT command FROM command WHERE soundId = $1", sound.ID)
		if err != nil {
			return nil, err
		}

		var commands []string
		for commandsRows.Next() {
			var command string
			if err := rows.Scan(&command); err != nil {
				return nil, err
			}

			commands = append(commands, command)
		}
		sound.Commands = commands

		sounds = append(sounds, &sound)
	}

	return sounds, nil
}

func getDB() (*sql.DB, error) {
	connPrefix := ""
	connSuffix := ""
	if config.DBDriver == "postgres" {
		connPrefix = "postgres://"

		ssl := "sslmode="
		if config.DBSSL {
			connSuffix = ssl + "verify-full"
		} else {
			connSuffix = ssl + "disable"
		}
	}

	connString := fmt.Sprintf("%s%s:%s@%s/%s?%s", connPrefix,
		config.DBUser, config.DBPassword, config.DBHost, config.DBName, connSuffix)
	return sql.Open(config.DBDriver, connString)
}

// UpdateSound update a sound in DB
func UpdateSound(gID string, sID string, s *Sound, commands []string) error {
	db, err := getDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	res, err := tx.Exec("INSERT INTO sound (guildID, name, gif, weight, filepath) VALUES ($1, $2, $3, $4, $5)",
		gID,
		s.Name,
		s.Gif,
		s.Weight)
	if err != nil {
		tx.Rollback()
		return err
	}
	soundID := res.LastInsertId

	_, err = tx.Exec("DELETE FROM command WHERE soundId = $1 AND guildId = $2")
	if err != nil {
		tx.Rollback()
		return err
	}

	for command := range commands {
		res, err = tx.Exec("INSERT INTO command (soundId, guildId, command) VALUES ($1, $2, $3)",
			soundID,
			gID,
			command)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// SaveSound saves a sound to the db
func SaveSound(gID string, s *Sound, commands []string) error {
	db, err := getDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	res, err := tx.Exec("INSERT INTO sound (guildID, name, gif, weight, filepath) VALUES ($1, $2, $3, $4, $5)",
		gID,
		s.Name,
		s.Gif,
		s.Weight)
	if err != nil {
		tx.Rollback()
		return err
	}
	soundID := res.LastInsertId

	for command := range commands {
		res, err = tx.Exec("INSERT INTO command (soundId, guildId, command) VALUES ($1, $2, $3)",
			soundID,
			gID,
			command)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// DeleteSound delete a sound from the DB
func DeleteSound(gID string, sID string) error {
	// TODO: delete also the sound file?
	db, err := getDB()
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FRON sound WHERE id = $1 AND guildId = $2", sID, gID)

	return err
}

// GetSoundsByCommand return all sounds for a given command
func GetSoundsByCommand(command, guildID string) ([]*Sound, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query("SELECT soundId FROM command WHERE guildId = $1 AND command = $2", guildID, command)
	if err != nil {
		return nil, err
	}

	var sounds []*Sound
	for rows.Next() {
		var soundID int
		if err := rows.Scan(&soundID); err != nil {
			return nil, err
		}

		row := db.QueryRow("SELECT id, name, gif, weight, filepath FROM sound WHERE id = $1", soundID)
		if err != nil {
			return nil, err
		}

		sound, err := buildSound(row)
		if err != nil {
			return nil, err
		}
		sounds = append(sounds, sound)
	}

	return sounds, nil
}

// GetSoundsByGuild return all sounds for a given Guild
func GetSoundsByGuild(guildID string) ([]*Sound, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query("SELECT * FROM sound WHERE guildId = $1", guildID)
	if err != nil {
		return nil, err
	}

	return buildSounds(db, rows)
}

// FilterByCommand filter a sound array by command
func FilterByCommand(c string, s []*Sound) (r []*Sound) {
	for _, sound := range s {
		for _, command := range sound.Commands {
			if c == command {
				r = append(r, sound)
				break
			}
		}
	}

	return r
}

// UtilGetRedisValuesFor keys
func UtilGetRedisValuesFor(redisPool *redis.Pool, keys []string) (r []interface{}, err error) {
	conn := redisPool.Get()
	for _, key := range keys {
		v, err := conn.Do("GET", key)
		if err != nil {
			return nil, err
		}
		r = append(r, v)
	}
	return r, nil
}

// DefaultSounds are a set of default sounds available to every servers
var DefaultSounds = []*Sound{
	&Sound{
		Name:     "airhorn_default",
		Weight:   1000,
		Commands: []string{"airhorn"},
		FilePath: "audio/airhorn_default.dca",
	},
	&Sound{
		Name:     "airhorn_reverb",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "audio/airhorn_reverb.dca",
	},
	&Sound{
		Name:     "airhorn_spam",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "audio/airhorn_spam.dca",
	},
	&Sound{
		Name:     "airhorn_tripletap",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "audio/airhorn_tripletap.dca",
	},
	&Sound{
		Name:     "airhorn_fourtap",
		Weight:   800,
		Commands: []string{"airhorn"},
		FilePath: "audio/airhorn_fourtap.dca",
	},
	&Sound{
		Name:     "airhorn_distant",
		Weight:   500,
		Commands: []string{"airhorn"},
		FilePath: "audio/airhorn_distant.dca",
	},
	&Sound{
		Name:     "airhorn_echo",
		Weight:   500,
		Commands: []string{"airhorn"},
		FilePath: "audio/airhorn_echo.dca",
	},
	&Sound{
		Name:     "airhorn_clownfull",
		Weight:   250,
		Commands: []string{"airhorn"},
		FilePath: "audio/airhorn_clownfull.dca",
	},
	&Sound{
		Name:     "airhorn_clownshort",
		Weight:   250,
		Commands: []string{"airhorn"},
		FilePath: "audio/airhorn_clownshort.dca",
	},
	&Sound{
		Name:     "airhorn_clownspam",
		Weight:   250,
		Commands: []string{"airhorn"},
		FilePath: "audio/airhorn_clownspam.dca",
	},
	&Sound{
		Name:     "airhorn_highfartlong",
		Weight:   200,
		Commands: []string{"airhorn"},
		FilePath: "audio/airhorn_highfartlong.dca",
	},
	&Sound{
		Name:     "airhorn_highfartshort",
		Weight:   200,
		Commands: []string{"airhorn"},
		FilePath: "audio/airhorn_highfartshot.dca",
	},
	&Sound{
		Name:     "airhorn_midshort",
		Weight:   100,
		Commands: []string{"airhorn"},
		FilePath: "audio/airhorn_midshort.dca",
	},
	&Sound{
		Name:     "airhorn_truck",
		Weight:   10,
		Commands: []string{"airhorn"},
		FilePath: "audio/airhorn_truck.dca",
	},
	&Sound{
		Name:     "another_one",
		Weight:   1,
		Commands: []string{"anotha"},
		FilePath: "audio/another_one.dca",
	},
	&Sound{
		Name:     "another_one_classic",
		Weight:   1,
		Commands: []string{"anotha"},
		FilePath: "audio/another_one_classic.dca",
	},
	&Sound{
		Name:     "another_one_echo",
		Weight:   1,
		Commands: []string{"anotha"},
		FilePath: "audio/another_one_echo.dca",
	},
	&Sound{
		Name:     "jc_realfull",
		Weight:   1,
		Commands: []string{"cena"},
		FilePath: "audio/jc_realfull.dca",
	},
	&Sound{
		Name:     "cow_herd",
		Weight:   10,
		Commands: []string{"stan"},
		FilePath: "audio/cow_herd.dca",
	},
	&Sound{
		Name:     "cow_moo",
		Weight:   10,
		Commands: []string{"stan"},
		FilePath: "audio/cow_moo.dca",
	},
	&Sound{
		Name:     "cow_x3",
		Weight:   1,
		Commands: []string{"stan"},
		FilePath: "audio/cow_x3.dca",
	},
	&Sound{
		Name:     "birthday_horn",
		Weight:   50,
		Commands: []string{"bday"},
		FilePath: "audio/birthday_horn.dca",
	},
	&Sound{
		Name:     "birthday_horn3",
		Weight:   30,
		Commands: []string{"bday"},
		FilePath: "audio/birthday_horn3.dca",
	},
	&Sound{
		Name:     "birthday_sadhorn",
		Weight:   25,
		Commands: []string{"bday"},
		FilePath: "audio/birthday_sadhorn.dca",
	},
	&Sound{
		Name:     "birthday_weakhorn",
		Weight:   25,
		Commands: []string{"bday"},
		FilePath: "audio/birthday_weakhorn.dca",
	},
	&Sound{
		Name:     "wow_thatscool",
		Weight:   1,
		Commands: []string{"wtc"},
		FilePath: "audio/wow_thatscool.dca",
	},
}
