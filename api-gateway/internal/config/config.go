package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Redis                    RedisConfig
	Port                     string
	MovieServiceURL          string
	UserPreferenceServiceURL string
	RecommendationServiceURL string
	RateLimitMax             int
	RateLimitWindowSeconds   int
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "3"))
	rateLimitMax, _ := strconv.Atoi(getEnv("RATE_LIMIT_MAX", "100"))
	rateLimitWindow, _ := strconv.Atoi(getEnv("RATE_LIMIT_WINDOW_SECONDS", "60"))

	return &Config{
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		Port:                     getEnv("SERVER_PORT", "8080"),
		MovieServiceURL:          getEnv("MOVIE_SERVICE_URL", "http://localhost:8081"),
		UserPreferenceServiceURL: getEnv("USER_PREFERENCE_SERVICE_URL", "http://localhost:8082"),
		RecommendationServiceURL: getEnv("RECOMMENDATION_SERVICE_URL", "http://localhost:8083"),
		RateLimitMax:             rateLimitMax,
		RateLimitWindowSeconds:   rateLimitWindow,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
