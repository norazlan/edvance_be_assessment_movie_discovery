package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Client is the TMDB API client.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewClient creates a new TMDB API client.
func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ---- TMDB Response Types (internal, not exposed to consumers) ----

// DiscoverResponse is the TMDB discover/movie response.
type DiscoverResponse struct {
	Page         int           `json:"page"`
	Results      []TMDBMovie   `json:"results"`
	TotalPages   int           `json:"total_pages"`
	TotalResults int           `json:"total_results"`
}

// TMDBMovie is a movie from TMDB discover results.
type TMDBMovie struct {
	ID               int     `json:"id"`
	Title            string  `json:"title"`
	Overview         string  `json:"overview"`
	ReleaseDate      string  `json:"release_date"`
	Popularity       float64 `json:"popularity"`
	PosterPath       string  `json:"poster_path"`
	BackdropPath     string  `json:"backdrop_path"`
	GenreIDs         []int   `json:"genre_ids"`
	OriginalLanguage string  `json:"original_language"`
}

// TMDBMovieDetail is the detailed movie info from TMDB.
type TMDBMovieDetail struct {
	ID               int         `json:"id"`
	Title            string      `json:"title"`
	Overview         string      `json:"overview"`
	ReleaseDate      string      `json:"release_date"`
	Popularity       float64     `json:"popularity"`
	PosterPath       string      `json:"poster_path"`
	BackdropPath     string      `json:"backdrop_path"`
	Genres           []TMDBGenre `json:"genres"`
	OriginalLanguage string      `json:"original_language"`
	Runtime          int         `json:"runtime"`
}

// TMDBGenre is a genre from TMDB.
type TMDBGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// GenreListResponse is the TMDB genre/movie/list response.
type GenreListResponse struct {
	Genres []TMDBGenre `json:"genres"`
}

// ---- Client Methods ----

// DiscoverMovies fetches movies from the TMDB discover endpoint.
func (c *Client) DiscoverMovies(page int) (*DiscoverResponse, error) {
	url := fmt.Sprintf(
		"%s/discover/movie?api_key=%s&sort_by=popularity.desc&page=%d",
		c.baseURL, c.apiKey, page,
	)

	slog.Debug("fetching TMDB discover", "url", url)
	resp, err := c.doGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result DiscoverResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode discover response: %w", err)
	}
	return &result, nil
}

// GetMovieDetail fetches detailed movie info from TMDB.
func (c *Client) GetMovieDetail(tmdbID int) (*TMDBMovieDetail, error) {
	url := fmt.Sprintf(
		"%s/movie/%d?api_key=%s",
		c.baseURL, tmdbID, c.apiKey,
	)

	slog.Debug("fetching TMDB movie detail", "tmdb_id", tmdbID)
	resp, err := c.doGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result TMDBMovieDetail
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode movie detail response: %w", err)
	}
	return &result, nil
}

// GetGenres fetches all movie genres from TMDB.
func (c *Client) GetGenres() ([]TMDBGenre, error) {
	url := fmt.Sprintf(
		"%s/genre/movie/list?api_key=%s",
		c.baseURL, c.apiKey,
	)

	slog.Debug("fetching TMDB genres")
	resp, err := c.doGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result GenreListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode genres response: %w", err)
	}
	return result.Genres, nil
}

func (c *Client) doGet(url string) (*http.Response, error) {
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("TMDB API returned status %d: %s", resp.StatusCode, string(body))
	}
	return resp, nil
}
