package redis

import (
	"context"

	"github.com/nurhudajoantama/hmauto/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

func NewClient(cfg config.Redis) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return rdb
}

func Close(ctx context.Context, rdb *redis.Client) {
	if err := rdb.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close redis connection")
	}
}
