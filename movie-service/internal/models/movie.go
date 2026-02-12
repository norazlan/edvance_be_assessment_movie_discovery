package models

import "time"

// Movie represents a movie stored in our database.
type Movie struct {
	ID               int       `json:"id"`
	TMDBId           int       `json:"tmdb_id"`
	Title            string    `json:"title"`
	Overview         string    `json:"overview"`
	ReleaseDate      string    `json:"release_date"`
	Popularity       float64   `json:"popularity"`
	PosterPath       string    `json:"poster_path"`
	BackdropPath     string    `json:"backdrop_path"`
	OriginalLanguage string    `json:"original_language"`
	Runtime          int       `json:"runtime"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Genre represents a movie genre.
type Genre struct {
	ID     int    `json:"id"`
	TMDBId int    `json:"tmdb_id"`
	Name   string `json:"name"`
}

// MovieListItem is the response shape for movie listing.
type MovieListItem struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	ReleaseDate string  `json:"release_date"`
	Popularity  float64 `json:"popularity"`
	PosterURL   string  `json:"poster_url"`
}

// MovieListResponse is the paginated movie listing response.
type MovieListResponse struct {
	Page         int             `json:"page"`
	PageSize     int             `json:"page_size"`
	TotalPages   int             `json:"total_pages"`
	TotalResults int             `json:"total_results"`
	Data         []MovieListItem `json:"data"`
}

// MovieDetail is the response shape for movie detail.
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
	BookingURL  string   `json:"booking_url"`
}

// MovieListParams holds query parameters for movie listing.
type MovieListParams struct {
	Page            int    `query:"page"`
	PageSize        int    `query:"page_size"`
	SortBy          string `query:"sort_by"`
	Order           string `query:"order"`
	ReleaseDateFrom string `query:"release_date_from"`
	ReleaseDateTo   string `query:"release_date_to"`
}

// Validate sets defaults and validates parameters.
func (p *MovieListParams) Validate() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 || p.PageSize > 100 {
		p.PageSize = 20
	}
	if p.SortBy == "" {
		p.SortBy = "popularity"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
	// Validate sort_by values
	validSorts := map[string]bool{"release_date": true, "title": true, "popularity": true}
	if !validSorts[p.SortBy] {
		p.SortBy = "popularity"
	}
	// Validate order values
	if p.Order != "asc" && p.Order != "desc" {
		p.Order = "desc"
	}
}

const (
	TMDBImageBaseW500 = "https://image.tmdb.org/t/p/w500"
	TMDBImageBaseW780 = "https://image.tmdb.org/t/p/w780"
	DefaultBookingURL = "https://www.google.com/"
)
