package service

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
)

var (
	// Redis client
	redisPool *redis.Pool
)

// InitRedis try to connect to redis using provided configuration.
// It does nothing if there's already a connection.
func InitRedis(cfg Cfg) bool {
	if redisPool != nil || cfg.RedisHost == "" {
		return false
	}

	// First, open a redis connection we use for stats
	if connectToRedis(cfg.RedisHost) != nil {
		return false
	}

	return true
}

// CloseRedis closes the redis connection
func CloseRedis() {
	if redisPool != nil {
		redisPool.Close()
	}
}

func connectToRedis(connStr string) (err error) {
	log.WithFields(log.Fields{
		"host": connStr,
	}).Info("Connecting to redis")

	redisPool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", connStr)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}

	// test redis connection
	conn := redisPool.Get()
	_, err = conn.Do("PING")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Can't establish a connection to the redis server")
		return err
	}
	conn.Close()

	return nil
}
