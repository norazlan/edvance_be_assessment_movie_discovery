package database

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/lib/pq"

	"movie-discovery-recommendation-service/internal/config"
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
		`CREATE TABLE IF NOT EXISTS recommendation_rules (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			weight DOUBLE PRECISION NOT NULL DEFAULT 1.0,
			rule_type VARCHAR(50) NOT NULL,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS user_recommendation_snapshots (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			movie_id INTEGER NOT NULL,
			score DOUBLE PRECISION NOT NULL,
			generated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(user_id, movie_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_recommendations_user_id ON user_recommendation_snapshots(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_recommendations_score ON user_recommendation_snapshots(score DESC)`,
		// Seed default rules if none exist
		`INSERT INTO recommendation_rules (name, weight, rule_type)
		 SELECT 'Popularity Score', 0.4, 'popularity'
		 WHERE NOT EXISTS (SELECT 1 FROM recommendation_rules WHERE rule_type = 'popularity')`,
		`INSERT INTO recommendation_rules (name, weight, rule_type)
		 SELECT 'Recency Bonus', 0.3, 'recency'
		 WHERE NOT EXISTS (SELECT 1 FROM recommendation_rules WHERE rule_type = 'recency')`,
		`INSERT INTO recommendation_rules (name, weight, rule_type)
		 SELECT 'Genre Match', 0.3, 'genre_match'
		 WHERE NOT EXISTS (SELECT 1 FROM recommendation_rules WHERE rule_type = 'genre_match')`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}

	slog.Info("database migrations completed")
	return nil
}
