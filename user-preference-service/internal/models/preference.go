package models

import "time"

// User represents a registered user.
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateUserRequest is the request body for creating a user.
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// UserPreference stores user preferences for movie recommendations.
type UserPreference struct {
	ID                int       `json:"id"`
	UserID            int       `json:"user_id"`
	PreferredGenres   []string  `json:"preferred_genres"`
	PreferredLanguage string    `json:"preferred_language"`
	MinRating         float64   `json:"min_rating"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// SetPreferenceRequest is the request body for setting preferences.
type SetPreferenceRequest struct {
	PreferredGenres   []string `json:"preferred_genres"`
	PreferredLanguage string   `json:"preferred_language"`
	MinRating         float64  `json:"min_rating"`
}

// UserInteraction records user activity with a movie.
type UserInteraction struct {
	ID              int       `json:"id"`
	UserID          int       `json:"user_id"`
	MovieID         int       `json:"movie_id"`
	InteractionType string    `json:"interaction_type"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateInteractionRequest is the request body for recording an interaction.
type CreateInteractionRequest struct {
	MovieID         int    `json:"movie_id"`
	InteractionType string `json:"interaction_type"`
}

// Valid interaction types
var ValidInteractionTypes = map[string]bool{
	"like":      true,
	"dislike":   true,
	"watchlist": true,
	"watched":   true,
}
