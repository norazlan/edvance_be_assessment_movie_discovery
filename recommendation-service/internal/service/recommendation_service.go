package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"movie-discovery-recommendation-service/internal/models"
	"movie-discovery-recommendation-service/internal/repository"
)

type RecommendationService struct {
	repo                     *repository.RecommendationRepository
	rdb                      *redis.Client
	movieServiceURL          string
	userPreferenceServiceURL string
	httpClient               *http.Client
}

func NewRecommendationService(
	repo *repository.RecommendationRepository,
	rdb *redis.Client,
	movieServiceURL, userPreferenceServiceURL string,
) *RecommendationService {
	return &RecommendationService{
		repo:                     repo,
		rdb:                      rdb,
		movieServiceURL:          strings.TrimRight(movieServiceURL, "/"),
		userPreferenceServiceURL: strings.TrimRight(userPreferenceServiceURL, "/"),
		httpClient:               &http.Client{Timeout: 15 * time.Second},
	}
}

// GetRecommendations generates personalized recommendations for a user.
func (s *RecommendationService) GetRecommendations(ctx context.Context, userID, limit int) (*models.RecommendationResponse, error) {
	// Check Redis cache first
	cacheKey := fmt.Sprintf("recommendations:%d:%d", userID, limit)
	if cached, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil {
		var resp models.RecommendationResponse
		if json.Unmarshal([]byte(cached), &resp) == nil {
			slog.Debug("recommendations cache hit", "user_id", userID)
			return &resp, nil
		}
	}

	// Fetch user preferences
	prefs, err := s.fetchUserPreferences(ctx, userID)
	if err != nil {
		slog.Warn("could not fetch user preferences, using defaults", "user_id", userID, "error", err)
		prefs = &models.UserPreference{
			UserID:          userID,
			PreferredGenres: []string{},
		}
	}

	// Fetch movies from movie service (multiple pages for better pool)
	allMovies, err := s.fetchMovies(ctx, 3)
	if err != nil {
		return nil, fmt.Errorf("fetch movies: %w", err)
	}

	if len(allMovies) == 0 {
		return &models.RecommendationResponse{
			UserID:          userID,
			Recommendations: []models.MovieRecommendation{},
			GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		}, nil
	}

	// Fetch active scoring rules
	rules, err := s.repo.GetActiveRules()
	if err != nil {
		return nil, fmt.Errorf("get rules: %w", err)
	}

	// Score each movie
	scored := s.scoreMovies(allMovies, prefs, rules)

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Limit results
	if len(scored) > limit {
		scored = scored[:limit]
	}

	// Persist snapshots asynchronously
	go func() {
		_ = s.repo.ClearSnapshots(userID)
		for _, rec := range scored {
			_ = s.repo.UpsertSnapshot(userID, rec.ID, rec.Score)
		}
	}()

	resp := &models.RecommendationResponse{
		UserID:          userID,
		Recommendations: scored,
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	// Cache for 10 minutes
	if data, err := json.Marshal(resp); err == nil {
		s.rdb.Set(ctx, cacheKey, data, 10*time.Minute)
	}

	return resp, nil
}

// scoreMovies applies weighted scoring rules to each movie.
func (s *RecommendationService) scoreMovies(
	movies []models.MovieDetail,
	prefs *models.UserPreference,
	rules []models.RecommendationRule,
) []models.MovieRecommendation {
	ruleWeights := make(map[string]float64)
	for _, r := range rules {
		ruleWeights[r.RuleType] = r.Weight
	}

	// Find max popularity for normalization
	var maxPop float64
	for _, m := range movies {
		if m.Popularity > maxPop {
			maxPop = m.Popularity
		}
	}
	if maxPop == 0 {
		maxPop = 1
	}

	prefGenreSet := make(map[string]bool)
	for _, g := range prefs.PreferredGenres {
		prefGenreSet[strings.ToLower(g)] = true
	}

	var results []models.MovieRecommendation
	for _, m := range movies {
		var totalScore float64
		var reasons []string

		// Popularity score (0–1 normalized)
		if w, ok := ruleWeights["popularity"]; ok {
			popScore := m.Popularity / maxPop
			totalScore += popScore * w
			if popScore > 0.7 {
				reasons = append(reasons, "highly popular")
			}
		}

		// Recency bonus (movies within the last 2 years get higher score)
		if w, ok := ruleWeights["recency"]; ok {
			recencyScore := computeRecencyScore(m.ReleaseDate)
			totalScore += recencyScore * w
			if recencyScore > 0.7 {
				reasons = append(reasons, "recently released")
			}
		}

		// Genre match
		if w, ok := ruleWeights["genre_match"]; ok && len(prefGenreSet) > 0 {
			genreScore := computeGenreMatchScore(m.Genres, prefGenreSet)
			totalScore += genreScore * w
			if genreScore > 0 {
				reasons = append(reasons, "matches your preferred genres")
			}
		}

		// Round score to 4 decimal places
		totalScore = math.Round(totalScore*10000) / 10000

		reason := "recommended for you"
		if len(reasons) > 0 {
			reason = strings.Join(reasons, ", ")
		}

		results = append(results, models.MovieRecommendation{
			ID:          m.ID,
			Title:       m.Title,
			ReleaseDate: m.ReleaseDate,
			Genres:      m.Genres,
			Popularity:  m.Popularity,
			PosterURL:   m.PosterURL,
			Score:       totalScore,
			Reason:      reason,
		})
	}

	return results
}

func computeRecencyScore(releaseDate string) float64 {
	t, err := time.Parse("2006-01-02", releaseDate)
	if err != nil {
		return 0.0
	}
	daysSince := time.Since(t).Hours() / 24
	if daysSince < 0 {
		daysSince = 0
	}
	// Score decays linearly over 730 days (2 years)
	score := 1.0 - (daysSince / 730.0)
	if score < 0 {
		score = 0
	}
	return score
}

func computeGenreMatchScore(movieGenres []string, preferredGenres map[string]bool) float64 {
	if len(movieGenres) == 0 {
		return 0.0
	}
	matches := 0
	for _, g := range movieGenres {
		if preferredGenres[strings.ToLower(g)] {
			matches++
		}
	}
	return float64(matches) / float64(len(movieGenres))
}

// fetchUserPreferences calls the user preference service.
func (s *RecommendationService) fetchUserPreferences(ctx context.Context, userID int) (*models.UserPreference, error) {
	url := fmt.Sprintf("%s/api/v1/users/%d/preferences", s.userPreferenceServiceURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to user-preference-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user-preference-service returned %d: %s", resp.StatusCode, string(body))
	}

	var prefs models.UserPreference
	if err := json.NewDecoder(resp.Body).Decode(&prefs); err != nil {
		return nil, fmt.Errorf("decode preferences: %w", err)
	}
	return &prefs, nil
}

// fetchMovies retrieves movies from the movie service.
func (s *RecommendationService) fetchMovies(ctx context.Context, pages int) ([]models.MovieDetail, error) {
	var allMovies []models.MovieDetail

	for page := 1; page <= pages; page++ {
		url := fmt.Sprintf("%s/api/v1/movies?page=%d&page_size=20&sort_by=popularity&order=desc", s.movieServiceURL, page)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request to movie-service page %d: %w", page, err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("movie-service returned %d: %s", resp.StatusCode, string(body))
		}

		var listResp models.MovieListResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode movie list: %w", err)
		}
		resp.Body.Close()

		// Fetch details for each movie to get genres
		for _, item := range listResp.Data {
			detail, err := s.fetchMovieDetail(ctx, item.ID)
			if err != nil {
				slog.Warn("could not fetch movie detail, using list data", "movie_id", item.ID, "error", err)
				allMovies = append(allMovies, models.MovieDetail{
					ID:          item.ID,
					Title:       item.Title,
					ReleaseDate: item.ReleaseDate,
					Popularity:  item.Popularity,
					PosterURL:   item.PosterURL,
				})
				continue
			}
			allMovies = append(allMovies, *detail)
		}

		if page >= listResp.TotalPages {
			break
		}
	}

	return allMovies, nil
}

func (s *RecommendationService) fetchMovieDetail(ctx context.Context, movieID int) (*models.MovieDetail, error) {
	url := fmt.Sprintf("%s/api/v1/movies/%d", s.movieServiceURL, movieID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("movie-service returned %d", resp.StatusCode)
	}

	var detail models.MovieDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

// GetRules returns all recommendation rules.
func (s *RecommendationService) GetRules(ctx context.Context) ([]models.RecommendationRule, error) {
	return s.repo.GetActiveRules()
}
