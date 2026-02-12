package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the movie service.
type Config struct {
	DB    DBConfig
	Redis RedisConfig
	TMDB  TMDBConfig
	Port  string
}

// DBConfig holds PostgreSQL configuration.
type DBConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	DBName      string
	SSLMode     string
	SSLRootCert string
}

// DSN returns the PostgreSQL connection string.
func (d DBConfig) DSN() string {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
	if d.SSLRootCert != "" {
		dsn += fmt.Sprintf(" sslrootcert=%s", d.SSLRootCert)
	}
	return dsn
}

// RedisConfig holds Redis configuration.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// TMDBConfig holds TMDB API configuration.
type TMDBConfig struct {
	APIKey  string
	BaseURL string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	cfg := &Config{
		DB: DBConfig{
			Host:        getEnv("DB_HOST", "localhost"),
			Port:        dbPort,
			User:        getEnv("DB_USER", "postgres"),
			Password:    getEnv("DB_PASSWORD", "postgres"),
			DBName:      getEnv("DB_NAME", "movie_service"),
			SSLMode:     getEnv("DB_SSLMODE", "verify-ca"),
			SSLRootCert: getEnv("DB_SSLROOTCERT", ""),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		TMDB: TMDBConfig{
			APIKey:  getEnv("TMDB_API_KEY", "XXXXXX"),
			BaseURL: getEnv("TMDB_BASE_URL", "http://api.themoviedb.org/3"),
		},
		Port: getEnv("SERVER_PORT", "8081"),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
