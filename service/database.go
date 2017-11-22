package service

import (
	"database/sql"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	// mysql driver, used via database/sql
	_ "github.com/go-sql-driver/mysql"
	// postgresql driver, used via database/sql
	_ "github.com/lib/pq"
)

var db *sql.DB

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

func getDB() (*sql.DB, error) {
  if (db != nil) {
    return db, nil
  }

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
