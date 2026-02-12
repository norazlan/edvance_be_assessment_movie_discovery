package repository

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"movie-discovery-movie-service/internal/models"
)

// MovieRepository handles database operations for movies.
type MovieRepository struct {
	db *sql.DB
}

// NewMovieRepository creates a new MovieRepository.
func NewMovieRepository(db *sql.DB) *MovieRepository {
	return &MovieRepository{db: db}
}

// UpsertGenre inserts or updates a genre.
func (r *MovieRepository) UpsertGenre(tmdbID int, name string) (int, error) {
	var id int
	err := r.db.QueryRow(`
		INSERT INTO genres (tmdb_id, name)
		VALUES ($1, $2)
		ON CONFLICT (tmdb_id) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, tmdbID, name).Scan(&id)
	return id, err
}

// UpsertMovie inserts or updates a movie.
func (r *MovieRepository) UpsertMovie(m *models.Movie) (int, error) {
	var id int
	err := r.db.QueryRow(`
		INSERT INTO movies (tmdb_id, title, overview, release_date, popularity,
			poster_path, backdrop_path, original_language, runtime, updated_at)
		VALUES ($1, $2, $3, $4::date, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (tmdb_id) DO UPDATE SET
			title = EXCLUDED.title,
			overview = EXCLUDED.overview,
			release_date = EXCLUDED.release_date,
			popularity = EXCLUDED.popularity,
			poster_path = EXCLUDED.poster_path,
			backdrop_path = EXCLUDED.backdrop_path,
			original_language = EXCLUDED.original_language,
			runtime = EXCLUDED.runtime,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`, m.TMDBId, m.Title, m.Overview, nullableDate(m.ReleaseDate),
		m.Popularity, m.PosterPath, m.BackdropPath,
		m.OriginalLanguage, m.Runtime, time.Now()).Scan(&id)
	return id, err
}

// LinkMovieGenre creates the movie-genre association.
func (r *MovieRepository) LinkMovieGenre(movieID, genreID int) error {
	_, err := r.db.Exec(`
		INSERT INTO movie_genres (movie_id, genre_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, movieID, genreID)
	return err
}

// GetGenreIDByTMDBId returns the internal genre ID for a TMDB genre ID.
func (r *MovieRepository) GetGenreIDByTMDBId(tmdbID int) (int, error) {
	var id int
	err := r.db.QueryRow(`SELECT id FROM genres WHERE tmdb_id = $1`, tmdbID).Scan(&id)
	return id, err
}

// ListMovies returns a paginated list of movies matching the given filters.
func (r *MovieRepository) ListMovies(params models.MovieListParams) (*models.MovieListResponse, error) {
	// Build WHERE clause
	conditions := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if params.ReleaseDateFrom != "" {
		conditions = append(conditions, fmt.Sprintf("m.release_date >= $%d::date", argIdx))
		args = append(args, params.ReleaseDateFrom)
		argIdx++
	}
	if params.ReleaseDateTo != "" {
		conditions = append(conditions, fmt.Sprintf("m.release_date <= $%d::date", argIdx))
		args = append(args, params.ReleaseDateTo)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Validate sort column to prevent SQL injection
	sortColumn := "popularity"
	switch params.SortBy {
	case "release_date":
		sortColumn = "release_date"
	case "title":
		sortColumn = "title"
	case "popularity":
		sortColumn = "popularity"
	}
	orderDir := "DESC"
	if params.Order == "asc" {
		orderDir = "ASC"
	}

	// Count total results
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM movies m WHERE %s", whereClause)
	var totalResults int
	if err := r.db.QueryRow(countQuery, args...).Scan(&totalResults); err != nil {
		return nil, fmt.Errorf("count query failed: %w", err)
	}

	// Calculate pagination
	offset := (params.Page - 1) * params.PageSize
	totalPages := 0
	if totalResults > 0 {
		totalPages = (totalResults + params.PageSize - 1) / params.PageSize
	}

	// Query movies
	listQuery := fmt.Sprintf(`
		SELECT m.id, m.title, 
			COALESCE(TO_CHAR(m.release_date, 'YYYY-MM-DD'), '') as release_date,
			m.popularity, COALESCE(m.poster_path, '') as poster_path
		FROM movies m
		WHERE %s
		ORDER BY m.%s %s NULLS LAST
		LIMIT $%d OFFSET $%d
	`, whereClause, sortColumn, orderDir, argIdx, argIdx+1)

	args = append(args, params.PageSize, offset)

	rows, err := r.db.Query(listQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list query failed: %w", err)
	}
	defer rows.Close()

	items := make([]models.MovieListItem, 0)
	for rows.Next() {
		var item models.MovieListItem
		var posterPath string
		if err := rows.Scan(&item.ID, &item.Title, &item.ReleaseDate, &item.Popularity, &posterPath); err != nil {
			slog.Error("failed to scan movie row", "error", err)
			continue
		}
		if posterPath != "" {
			item.PosterURL = models.TMDBImageBaseW500 + posterPath
		}
		items = append(items, item)
	}

	return &models.MovieListResponse{
		Page:         params.Page,
		PageSize:     params.PageSize,
		TotalPages:   totalPages,
		TotalResults: totalResults,
		Data:         items,
	}, nil
}

// GetMovieByID returns detailed movie information by internal ID.
func (r *MovieRepository) GetMovieByID(id int) (*models.MovieDetail, error) {
	var detail models.MovieDetail
	var posterPath, backdropPath string

	err := r.db.QueryRow(`
		SELECT m.id, m.title, COALESCE(m.overview, ''),
			COALESCE(TO_CHAR(m.release_date, 'YYYY-MM-DD'), ''),
			m.original_language, m.runtime, m.popularity,
			COALESCE(m.poster_path, ''), COALESCE(m.backdrop_path, '')
		FROM movies m
		WHERE m.id = $1
	`, id).Scan(
		&detail.ID, &detail.Title, &detail.Overview,
		&detail.ReleaseDate, &detail.Language, &detail.Duration,
		&detail.Popularity, &posterPath, &backdropPath,
	)
	if err != nil {
		return nil, err
	}

	if posterPath != "" {
		detail.PosterURL = models.TMDBImageBaseW500 + posterPath
	}
	if backdropPath != "" {
		detail.BackdropURL = models.TMDBImageBaseW780 + backdropPath
	}
	detail.BookingURL = models.DefaultBookingURL

	// Fetch genres
	rows, err := r.db.Query(`
		SELECT g.name FROM genres g
		INNER JOIN movie_genres mg ON mg.genre_id = g.id
		WHERE mg.movie_id = $1
		ORDER BY g.name
	`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query genres: %w", err)
	}
	defer rows.Close()

	detail.Genres = make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			detail.Genres = append(detail.Genres, name)
		}
	}

	return &detail, nil
}

// GetMovieByTMDBId returns detailed movie information by TMDB ID.
func (r *MovieRepository) GetMovieByTMDBId(tmdbID int) (*models.MovieDetail, error) {
	var internalID int
	err := r.db.QueryRow(`SELECT id FROM movies WHERE tmdb_id = $1`, tmdbID).Scan(&internalID)
	if err != nil {
		return nil, err
	}
	return r.GetMovieByID(internalID)
}

// ClearMovieGenres removes all genre links for a movie.
func (r *MovieRepository) ClearMovieGenres(movieID int) error {
	_, err := r.db.Exec(`DELETE FROM movie_genres WHERE movie_id = $1`, movieID)
	return err
}

// GetAllMovies returns all movie IDs and TMDB IDs (for syncing runtime).
func (r *MovieRepository) GetAllMovies() ([]struct{ ID, TMDBId int }, error) {
	rows, err := r.db.Query(`SELECT id, tmdb_id FROM movies WHERE runtime = 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []struct{ ID, TMDBId int }
	for rows.Next() {
		var item struct{ ID, TMDBId int }
		if err := rows.Scan(&item.ID, &item.TMDBId); err == nil {
			result = append(result, item)
		}
	}
	return result, nil
}

// UpdateRuntime sets the runtime for a movie.
func (r *MovieRepository) UpdateRuntime(id, runtime int) error {
	_, err := r.db.Exec(`UPDATE movies SET runtime = $1, updated_at = NOW() WHERE id = $2`, runtime, id)
	return err
}

func nullableDate(dateStr string) interface{} {
	if dateStr == "" {
		return nil
	}
	return dateStr
}
