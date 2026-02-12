package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"

	"movie-discovery-recommendation-service/internal/config"
)

func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	slog.Info("connected to Redis", "addr", cfg.Addr)
	return client, nil
}
