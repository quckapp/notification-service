package db

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/quickchat/notification-service/internal/config"
)

// Redis wraps the Redis client
type Redis struct {
	*redis.Client
}

// NewRedis creates a new Redis connection
func NewRedis(cfg config.RedisConfig) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     20,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return &Redis{client}, nil
}

// Close closes the Redis connection
func (r *Redis) Close() error {
	return r.Client.Close()
}
