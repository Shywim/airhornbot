package service

import (
	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	// mysql driver, used via database/sql
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	// postgresql driver, used via database/sql
	_ "github.com/lib/pq"
)

var db *sqlx.DB

func initDb() {
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
	var err error
	db, err = sqlx.Open(config.DBDriver, connString)
	if err != nil {
		log.WithError(err).Fatal("Unable to connect to database")
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
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Error creating tables")
	}

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

func getDB() *sqlx.DB {
	return db
}

func insertGetID(d sqlx.Ext, query string, args ...interface{}) (string, error) {
	var id int64
	if config.DBDriver == "postgres" {
		pgQuery := query + " RETURNING id"
		res := d.QueryRowx(pgQuery, args...)

		err := res.Scan(&id)
		if err != nil {
			return "", err
		}
	} else {
		res, err := d.Exec(query, args...)
		if err != nil {
			return "", err
		}

		id, err = res.LastInsertId()
		if err != nil {
			return "", err
		}
	}

	return strconv.FormatInt(id, 10), nil
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
