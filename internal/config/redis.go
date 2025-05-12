package config

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisConf struct {
	Host string `json:"host"`
	Type string `json:",default=node,options=node|cluster"`
	Pass string `json:",optional"`
	Tls  bool   `json:",optional"`
	DB   int    `json:",default=0"`
}

func InitRedis(redisConf RedisConf) *redis.Client {
	myRedis := redis.NewClient(&redis.Options{
		Addr:     redisConf.Host,
		Password: redisConf.Pass,
		DB:       0,
	})

	_, err := myRedis.Ping(context.Background()).Result()
	if err != nil {
		panic("redis connect failed: " + err.Error())
	}
	return myRedis
}
