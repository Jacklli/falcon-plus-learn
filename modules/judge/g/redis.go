package g

import (
	"github.com/garyburd/redigo/redis"
	"log"
	"time"
)

var RedisConnPool *redis.Pool

/*
初始化redis连接池
 */
func InitRedisConnPool() {
	if !Config().Alarm.Enabled { // 判断是否开启alarm
		return
	}

	dsn := Config().Alarm.Redis.Dsn
	maxIdle := Config().Alarm.Redis.MaxIdle
	idleTimeout := 240 * time.Second

	connTimeout := time.Duration(Config().Alarm.Redis.ConnTimeout) * time.Millisecond
	readTimeout := time.Duration(Config().Alarm.Redis.ReadTimeout) * time.Millisecond
	writeTimeout := time.Duration(Config().Alarm.Redis.WriteTimeout) * time.Millisecond

	// 创建redis连接池, 使用了https://godoc.org/github.com/garyburd/redigo/redis
	RedisConnPool = &redis.Pool{
		MaxIdle:     maxIdle,
		IdleTimeout: idleTimeout,
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialTimeout("tcp", dsn, connTimeout, readTimeout, writeTimeout)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: PingRedis,
	}
}

func PingRedis(c redis.Conn, t time.Time) error {
	_, err := c.Do("ping")
	if err != nil {
		log.Println("[ERROR] ping redis fail", err)
	}
	return err
}
