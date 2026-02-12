package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"movie-discovery-movie-service/internal/models"
	"movie-discovery-movie-service/internal/repository"
	"movie-discovery-movie-service/internal/tmdb"
)

const (
	movieListCacheTTL   = 5 * time.Minute
	movieDetailCacheTTL = 30 * time.Minute
)

// MovieService handles business logic for movies.
type MovieService struct {
	repo       *repository.MovieRepository
	tmdbClient *tmdb.Client
	redis      *redis.Client
}

// NewMovieService creates a new MovieService.
func NewMovieService(repo *repository.MovieRepository, tmdbClient *tmdb.Client, rdb *redis.Client) *MovieService {
	return &MovieService{
		repo:       repo,
		tmdbClient: tmdbClient,
		redis:      rdb,
	}
}

// SyncMovies fetches movies from TMDB and stores them in PostgreSQL.
func (s *MovieService) SyncMovies(pages int) (int, error) {
	slog.Info("starting TMDB sync", "pages", pages)

	// First, sync genres
	genres, err := s.tmdbClient.GetGenres()
	if err != nil {
		return 0, fmt.Errorf("failed to fetch TMDB genres: %w", err)
	}
	for _, g := range genres {
		if _, err := s.repo.UpsertGenre(g.ID, g.Name); err != nil {
			slog.Error("failed to upsert genre", "genre", g.Name, "error", err)
		}
	}
	slog.Info("synced genres", "count", len(genres))

	// Then, sync movies from discover endpoint
	totalSynced := 0
	for page := 1; page <= pages; page++ {
		result, err := s.tmdbClient.DiscoverMovies(page)
		if err != nil {
			slog.Error("failed to fetch TMDB page", "page", page, "error", err)
			continue
		}

		for _, tmdbMovie := range result.Results {
			movie := &models.Movie{
				TMDBId:           tmdbMovie.ID,
				Title:            tmdbMovie.Title,
				Overview:         tmdbMovie.Overview,
				ReleaseDate:      tmdbMovie.ReleaseDate,
				Popularity:       tmdbMovie.Popularity,
				PosterPath:       tmdbMovie.PosterPath,
				BackdropPath:     tmdbMovie.BackdropPath,
				OriginalLanguage: tmdbMovie.OriginalLanguage,
			}

			movieID, err := s.repo.UpsertMovie(movie)
			if err != nil {
				slog.Error("failed to upsert movie", "title", movie.Title, "error", err)
				continue
			}

			// Clear existing genre links and re-create
			_ = s.repo.ClearMovieGenres(movieID)
			for _, genreID := range tmdbMovie.GenreIDs {
				internalGenreID, err := s.repo.GetGenreIDByTMDBId(genreID)
				if err != nil {
					continue
				}
				_ = s.repo.LinkMovieGenre(movieID, internalGenreID)
			}

			totalSynced++
		}

		slog.Info("synced page", "page", page, "movies", len(result.Results))
	}

	// Fetch runtime for movies that don't have it yet
	go s.syncRuntimes()

	// Invalidate Redis cache after sync
	s.invalidateCache()

	slog.Info("TMDB sync completed", "total_synced", totalSynced)
	return totalSynced, nil
}

// syncRuntimes fetches runtime for movies that don't have it.
func (s *MovieService) syncRuntimes() {
	movies, err := s.repo.GetAllMovies()
	if err != nil {
		slog.Error("failed to get movies for runtime sync", "error", err)
		return
	}

	for _, m := range movies {
		detail, err := s.tmdbClient.GetMovieDetail(m.TMDBId)
		if err != nil {
			slog.Error("failed to fetch movie detail", "tmdb_id", m.TMDBId, "error", err)
			continue
		}
		if err := s.repo.UpdateRuntime(m.ID, detail.Runtime); err != nil {
			slog.Error("failed to update runtime", "id", m.ID, "error", err)
		}
		// Rate limit TMDB requests
		time.Sleep(100 * time.Millisecond)
	}
	slog.Info("runtime sync completed", "count", len(movies))
}

// ListMovies returns a paginated list of movies.
func (s *MovieService) ListMovies(params models.MovieListParams) (*models.MovieListResponse, error) {
	params.Validate()

	// Try Redis cache
	cacheKey := fmt.Sprintf("movies:list:%d:%d:%s:%s:%s:%s",
		params.Page, params.PageSize, params.SortBy, params.Order,
		params.ReleaseDateFrom, params.ReleaseDateTo)

	if cached, err := s.getFromCache(cacheKey); err == nil {
		var result models.MovieListResponse
		if json.Unmarshal([]byte(cached), &result) == nil {
			slog.Debug("cache hit", "key", cacheKey)
			return &result, nil
		}
	}

	// Query from database
	result, err := s.repo.ListMovies(params)
	if err != nil {
		return nil, fmt.Errorf("failed to list movies: %w", err)
	}

	// Store in cache
	if data, err := json.Marshal(result); err == nil {
		s.setCache(cacheKey, string(data), movieListCacheTTL)
	}

	return result, nil
}

// GetMovieDetail returns detailed movie info by ID.
func (s *MovieService) GetMovieDetail(id int) (*models.MovieDetail, error) {
	// Try Redis cache
	cacheKey := fmt.Sprintf("movie:detail:%d", id)

	if cached, err := s.getFromCache(cacheKey); err == nil {
		var result models.MovieDetail
		if json.Unmarshal([]byte(cached), &result) == nil {
			slog.Debug("cache hit", "key", cacheKey)
			return &result, nil
		}
	}

	// Query from database
	detail, err := s.repo.GetMovieByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("movie not found")
		}
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	// Store in cache
	if data, err := json.Marshal(detail); err == nil {
		s.setCache(cacheKey, string(data), movieDetailCacheTTL)
	}

	return detail, nil
}

// ---- Redis Helpers ----

func (s *MovieService) getFromCache(key string) (string, error) {
	if s.redis == nil {
		return "", fmt.Errorf("redis not available")
	}
	return s.redis.Get(context.Background(), key).Result()
}

func (s *MovieService) setCache(key, value string, ttl time.Duration) {
	if s.redis == nil {
		return
	}
	if err := s.redis.Set(context.Background(), key, value, ttl).Err(); err != nil {
		slog.Error("failed to set cache", "key", key, "error", err)
	}
}

func (s *MovieService) invalidateCache() {
	if s.redis == nil {
		return
	}
	ctx := context.Background()
	iter := s.redis.Scan(ctx, 0, "movies:*", 0).Iterator()
	for iter.Next(ctx) {
		s.redis.Del(ctx, iter.Val())
	}
	iter2 := s.redis.Scan(ctx, 0, "movie:*", 0).Iterator()
	for iter2.Next(ctx) {
		s.redis.Del(ctx, iter2.Val())
	}
	slog.Info("Redis cache invalidated")
}
