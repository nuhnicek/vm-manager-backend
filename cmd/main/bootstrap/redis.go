package bootstrap

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis(redisURI string) (error, *redis.Client) {
	opt, err := redis.ParseURL(redisURI)
	if err != nil {
		return err, nil
	}
	client := redis.NewClient(opt)
	err = client.Ping(context.Background()).Err()
	if err != nil {
		return err, nil
	}
	return nil, client
}
