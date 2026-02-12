package models

import "time"

// RecommendationRule defines a scoring rule.
type RecommendationRule struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Weight    float64   `json:"weight"`
	RuleType  string    `json:"rule_type"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// RecommendationSnapshot stores a computed recommendation.
type RecommendationSnapshot struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	MovieID     int       `json:"movie_id"`
	Score       float64   `json:"score"`
	GeneratedAt time.Time `json:"generated_at"`
}

// MovieRecommendation is the response shape for a recommended movie.
type MovieRecommendation struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	ReleaseDate string   `json:"release_date"`
	Genres      []string `json:"genres"`
	Popularity  float64  `json:"popularity"`
	PosterURL   string   `json:"poster_url"`
	Score       float64  `json:"score"`
	Reason      string   `json:"reason"`
}

// RecommendationResponse wraps the recommendation list.
type RecommendationResponse struct {
	UserID          int                   `json:"user_id"`
	Recommendations []MovieRecommendation `json:"recommendations"`
	GeneratedAt     string                `json:"generated_at"`
}

// MovieListItem represents a movie from the movie service.
type MovieListItem struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	ReleaseDate string  `json:"release_date"`
	Popularity  float64 `json:"popularity"`
	PosterURL   string  `json:"poster_url"`
}

// MovieListResponse represents the movie service list response.
type MovieListResponse struct {
	Page         int             `json:"page"`
	PageSize     int             `json:"page_size"`
	TotalPages   int             `json:"total_pages"`
	TotalResults int             `json:"total_results"`
	Data         []MovieListItem `json:"data"`
}

// MovieDetail represents movie detail from the movie service.
type MovieDetail struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Overview    string   `json:"overview"`
	ReleaseDate string   `json:"release_date"`
	Genres      []string `json:"genres"`
	Language    string   `json:"language"`
	Duration    int      `json:"duration"`
	Popularity  float64  `json:"popularity"`
	PosterURL   string   `json:"poster_url"`
	BackdropURL string   `json:"backdrop_url"`
}

// UserPreference represents preferences from the user preference service.
type UserPreference struct {
	UserID            int      `json:"user_id"`
	PreferredGenres   []string `json:"preferred_genres"`
	PreferredLanguage string   `json:"preferred_language"`
	MinRating         float64  `json:"min_rating"`
}
