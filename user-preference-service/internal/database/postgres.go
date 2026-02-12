package database

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/lib/pq"

	"movie-discovery-user-preference-service/internal/config"
)

func NewPostgres(cfg config.DBConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)

	slog.Info("connected to PostgreSQL", "db", cfg.DBName)

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(100) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS user_preferences (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			preferred_genres TEXT[] DEFAULT '{}',
			preferred_language VARCHAR(10) DEFAULT 'en',
			min_rating DOUBLE PRECISION DEFAULT 0,
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_interactions (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			movie_id INTEGER NOT NULL,
			interaction_type VARCHAR(50) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_interactions_user_id ON user_interactions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_interactions_movie_id ON user_interactions(movie_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_preferences_user_id ON user_preferences(user_id)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}

	slog.Info("database migrations completed")
	return nil
}
