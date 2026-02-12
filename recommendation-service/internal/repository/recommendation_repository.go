package repository

import (
	"database/sql"
	"fmt"

	"movie-discovery-recommendation-service/internal/models"
)

type RecommendationRepository struct {
	db *sql.DB
}

func NewRecommendationRepository(db *sql.DB) *RecommendationRepository {
	return &RecommendationRepository{db: db}
}

// GetActiveRules returns all active recommendation rules.
func (r *RecommendationRepository) GetActiveRules() ([]models.RecommendationRule, error) {
	rows, err := r.db.Query(`
		SELECT id, name, weight, rule_type, is_active, created_at
		FROM recommendation_rules
		WHERE is_active = TRUE
		ORDER BY rule_type
	`)
	if err != nil {
		return nil, fmt.Errorf("query active rules: %w", err)
	}
	defer rows.Close()

	var rules []models.RecommendationRule
	for rows.Next() {
		var rule models.RecommendationRule
		if err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Weight,
			&rule.RuleType, &rule.IsActive, &rule.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

// UpsertSnapshot stores or updates a recommendation score snapshot.
func (r *RecommendationRepository) UpsertSnapshot(userID, movieID int, score float64) error {
	_, err := r.db.Exec(`
		INSERT INTO user_recommendation_snapshots (user_id, movie_id, score, generated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, movie_id)
		DO UPDATE SET score = EXCLUDED.score, generated_at = NOW()
	`, userID, movieID, score)
	if err != nil {
		return fmt.Errorf("upsert snapshot: %w", err)
	}
	return nil
}

// GetSnapshots retrieves the top N recommendation snapshots for a user.
func (r *RecommendationRepository) GetSnapshots(userID, limit int) ([]models.RecommendationSnapshot, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, movie_id, score, generated_at
		FROM user_recommendation_snapshots
		WHERE user_id = $1
		ORDER BY score DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []models.RecommendationSnapshot
	for rows.Next() {
		var s models.RecommendationSnapshot
		if err := rows.Scan(&s.ID, &s.UserID, &s.MovieID, &s.Score, &s.GeneratedAt); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, rows.Err()
}

// ClearSnapshots removes all snapshots for a user (before regeneration).
func (r *RecommendationRepository) ClearSnapshots(userID int) error {
	_, err := r.db.Exec(`DELETE FROM user_recommendation_snapshots WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("clear snapshots: %w", err)
	}
	return nil
}
