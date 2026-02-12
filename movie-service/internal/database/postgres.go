package database

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/lib/pq"

	"movie-discovery-movie-service/internal/config"
)

// NewPostgres creates a new PostgreSQL connection and runs migrations.
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
		`CREATE TABLE IF NOT EXISTS genres (
			id SERIAL PRIMARY KEY,
			tmdb_id INTEGER UNIQUE NOT NULL,
			name VARCHAR(100) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS movies (
			id SERIAL PRIMARY KEY,
			tmdb_id INTEGER UNIQUE NOT NULL,
			title VARCHAR(500) NOT NULL,
			overview TEXT DEFAULT '',
			release_date DATE,
			popularity DOUBLE PRECISION DEFAULT 0,
			poster_path VARCHAR(500) DEFAULT '',
			backdrop_path VARCHAR(500) DEFAULT '',
			original_language VARCHAR(10) DEFAULT 'en',
			runtime INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS movie_genres (
			movie_id INTEGER REFERENCES movies(id) ON DELETE CASCADE,
			genre_id INTEGER REFERENCES genres(id) ON DELETE CASCADE,
			PRIMARY KEY (movie_id, genre_id)
		)`,
		// Indexes for common query patterns
		`CREATE INDEX IF NOT EXISTS idx_movies_release_date ON movies(release_date)`,
		`CREATE INDEX IF NOT EXISTS idx_movies_popularity ON movies(popularity)`,
		`CREATE INDEX IF NOT EXISTS idx_movies_title ON movies(title)`,
		`CREATE INDEX IF NOT EXISTS idx_movies_tmdb_id ON movies(tmdb_id)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}

	slog.Info("database migrations completed")
	return nil
}
