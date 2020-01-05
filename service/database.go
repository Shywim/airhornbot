package service

import (
	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"

	"github.com/jmoiron/sqlx"

	// postgresql driver, used via database/sql
	_ "github.com/lib/pq"

	migrate "github.com/rubenv/sql-migrate"
)

var db *sqlx.DB

func initDb() {
	connPrefix := "postgres://"
	connSuffix := ""

	ssl := "sslmode="
	if config.DBSSL {
		connSuffix = ssl + "verify-full"
	} else {
		connSuffix = ssl + "disable"
	}

	connString := fmt.Sprintf("%s%s:%s@%s/%s?%s", connPrefix,
		config.DBUser, config.DBPassword, config.DBHost, config.DBName, connSuffix)
	var err error
	db, err = sqlx.Open(config.DBDriver, connString)
	if err != nil {
		log.WithError(err).Fatal("Unable to connect to database")
		return
	}

	migrations := &migrate.FileMigrationSource{
		Dir: "db/migrations",
	}
	n, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	if err != nil {
		log.WithError(err).Fatal("Failed to execute database migrations")
	}
	log.WithFields(log.Fields{
		"migrations": n,
	}).Info("Successfully applied database migrations")
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
