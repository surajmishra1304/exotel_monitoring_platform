package database

import (
	"context"
	"fmt"

	"exotel-monitoring-platform/internal/config"
	"exotel-monitoring-platform/internal/logger"

	"github.com/go-redis/redis/v8"
)

var Redis *redis.Client
var Ctx = context.Background()

// ConnectRedis creates a Redis client and verifies the connection with PING.
func ConnectRedis() {
	cfg := config.App.Redis

	Redis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	if err := Redis.Ping(Ctx).Err(); err != nil {
		panic(fmt.Errorf("redis connection failed: %w", err))
	}

	logger.Log.Info("Redis connected successfully")
}
