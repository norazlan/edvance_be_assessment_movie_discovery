package repository

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"movie-discovery-user-preference-service/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser creates a new user.
func (r *UserRepository) CreateUser(req models.CreateUserRequest) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(`
		INSERT INTO users (username, email) VALUES ($1, $2)
		RETURNING id, username, email, created_at
	`, req.Username, req.Email).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

// GetUser returns a user by ID.
func (r *UserRepository) GetUser(id int) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(`
		SELECT id, username, email, created_at FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpsertPreference creates or updates user preferences.
func (r *UserRepository) UpsertPreference(userID int, req models.SetPreferenceRequest) (*models.UserPreference, error) {
	var pref models.UserPreference
	err := r.db.QueryRow(`
		INSERT INTO user_preferences (user_id, preferred_genres, preferred_language, min_rating, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			preferred_genres = EXCLUDED.preferred_genres,
			preferred_language = EXCLUDED.preferred_language,
			min_rating = EXCLUDED.min_rating,
			updated_at = NOW()
		RETURNING id, user_id, preferred_genres, preferred_language, min_rating, updated_at
	`, userID, pq.Array(req.PreferredGenres), req.PreferredLanguage, req.MinRating).Scan(
		&pref.ID, &pref.UserID, pq.Array(&pref.PreferredGenres),
		&pref.PreferredLanguage, &pref.MinRating, &pref.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert preference: %w", err)
	}
	return &pref, nil
}

// GetPreference returns user preferences.
func (r *UserRepository) GetPreference(userID int) (*models.UserPreference, error) {
	var pref models.UserPreference
	err := r.db.QueryRow(`
		SELECT id, user_id, preferred_genres, preferred_language, min_rating, updated_at
		FROM user_preferences WHERE user_id = $1
	`, userID).Scan(
		&pref.ID, &pref.UserID, pq.Array(&pref.PreferredGenres),
		&pref.PreferredLanguage, &pref.MinRating, &pref.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &pref, nil
}

// CreateInteraction records a user interaction.
func (r *UserRepository) CreateInteraction(userID int, req models.CreateInteractionRequest) (*models.UserInteraction, error) {
	var inter models.UserInteraction
	err := r.db.QueryRow(`
		INSERT INTO user_interactions (user_id, movie_id, interaction_type)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, movie_id, interaction_type, created_at
	`, userID, req.MovieID, req.InteractionType).Scan(
		&inter.ID, &inter.UserID, &inter.MovieID, &inter.InteractionType, &inter.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create interaction: %w", err)
	}
	return &inter, nil
}

// GetInteractions returns interactions for a user.
func (r *UserRepository) GetInteractions(userID int, limit int) ([]models.UserInteraction, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, movie_id, interaction_type, created_at
		FROM user_interactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query interactions: %w", err)
	}
	defer rows.Close()

	var interactions []models.UserInteraction
	for rows.Next() {
		var inter models.UserInteraction
		if err := rows.Scan(&inter.ID, &inter.UserID, &inter.MovieID, &inter.InteractionType, &inter.CreatedAt); err != nil {
			continue
		}
		interactions = append(interactions, inter)
	}
	return interactions, nil
}
