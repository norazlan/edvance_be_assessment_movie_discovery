package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DB                     DBConfig
	Redis                  RedisConfig
	Port                   string
	MovieServiceURL        string
	UserPreferenceServiceURL string
}

type DBConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	DBName      string
	SSLMode     string
	SSLRootCert string
}

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

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "2"))

	return &Config{
		DB: DBConfig{
			Host:        getEnv("DB_HOST", "localhost"),
			Port:        dbPort,
			User:        getEnv("DB_USER", "postgres"),
			Password:    getEnv("DB_PASSWORD", "postgres"),
			DBName:      getEnv("DB_NAME", "recommendation_service"),
			SSLMode:     getEnv("DB_SSLMODE", "verify-ca"),
			SSLRootCert: getEnv("DB_SSLROOTCERT", ""),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		Port:                     getEnv("SERVER_PORT", "8083"),
		MovieServiceURL:          getEnv("MOVIE_SERVICE_URL", "http://localhost:8081"),
		UserPreferenceServiceURL: getEnv("USER_PREFERENCE_SERVICE_URL", "http://localhost:8082"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
