package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"movie-discovery-api-gateway/internal/config"
)

// NewRedisClient creates a redis client for the gateway.
func NewRedisClient(cfg config.RedisConfig) (*redis.Client, error) {
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

// RateLimiter provides Redis-backed sliding window rate limiting.
type RateLimiter struct {
	rdb       *redis.Client
	maxReqs   int
	windowSec int
}

// NewRateLimiter creates a rate limiter.
func NewRateLimiter(rdb *redis.Client, maxReqs, windowSec int) *RateLimiter {
	return &RateLimiter{
		rdb:       rdb,
		maxReqs:   maxReqs,
		windowSec: windowSec,
	}
}

// Handler returns a Fiber middleware handler for rate limiting.
func (rl *RateLimiter) Handler() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Use client IP as the rate limit key
		ip := c.IP()
		key := fmt.Sprintf("ratelimit:%s", ip)
		ctx := context.Background()

		// Increment counter
		count, err := rl.rdb.Incr(ctx, key).Result()
		if err != nil {
			// If Redis fails, allow the request (fail-open)
			return c.Next()
		}

		// Set expiry on first request in the window
		if count == 1 {
			rl.rdb.Expire(ctx, key, time.Duration(rl.windowSec)*time.Second)
		}

		// Get remaining TTL for headers
		ttl, _ := rl.rdb.TTL(ctx, key).Result()

		// Set rate limit headers
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.maxReqs))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, int64(rl.maxReqs)-count)))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", int(ttl.Seconds())))

		if int(count) > rl.maxReqs {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "rate limit exceeded",
				"retry_after": int(ttl.Seconds()),
			})
		}

		return c.Next()
	}
}
