package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"movie-discovery-user-preference-service/internal/models"
	"movie-discovery-user-preference-service/internal/repository"
)

const (
	prefCacheTTL = 10 * time.Minute
)

type UserService struct {
	repo  *repository.UserRepository
	redis *redis.Client
}

func NewUserService(repo *repository.UserRepository, rdb *redis.Client) *UserService {
	return &UserService{repo: repo, redis: rdb}
}

func (s *UserService) CreateUser(req models.CreateUserRequest) (*models.User, error) {
	if req.Username == "" || req.Email == "" {
		return nil, fmt.Errorf("username and email are required")
	}
	return s.repo.CreateUser(req)
}

func (s *UserService) GetUser(id int) (*models.User, error) {
	user, err := s.repo.GetUser(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return user, nil
}

func (s *UserService) SetPreference(userID int, req models.SetPreferenceRequest) (*models.UserPreference, error) {
	// Verify user exists
	if _, err := s.repo.GetUser(userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	pref, err := s.repo.UpsertPreference(userID, req)
	if err != nil {
		return nil, err
	}

	// Invalidate cache
	s.delCache(fmt.Sprintf("user:pref:%d", userID))

	return pref, nil
}

func (s *UserService) GetPreference(userID int) (*models.UserPreference, error) {
	// Try cache
	cacheKey := fmt.Sprintf("user:pref:%d", userID)
	if cached, err := s.getFromCache(cacheKey); err == nil {
		var pref models.UserPreference
		if json.Unmarshal([]byte(cached), &pref) == nil {
			return &pref, nil
		}
	}

	pref, err := s.repo.GetPreference(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default preferences
			return &models.UserPreference{
				UserID:            userID,
				PreferredGenres:   []string{},
				PreferredLanguage: "en",
				MinRating:         0,
			}, nil
		}
		return nil, err
	}

	// Cache result
	if data, err := json.Marshal(pref); err == nil {
		s.setCache(cacheKey, string(data), prefCacheTTL)
	}

	return pref, nil
}

func (s *UserService) RecordInteraction(userID int, req models.CreateInteractionRequest) (*models.UserInteraction, error) {
	if !models.ValidInteractionTypes[req.InteractionType] {
		return nil, fmt.Errorf("invalid interaction type: %s", req.InteractionType)
	}
	if req.MovieID <= 0 {
		return nil, fmt.Errorf("invalid movie ID")
	}

	// Verify user exists
	if _, err := s.repo.GetUser(userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	return s.repo.CreateInteraction(userID, req)
}

func (s *UserService) GetInteractions(userID, limit int) ([]models.UserInteraction, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.GetInteractions(userID, limit)
}

// Redis helpers

func (s *UserService) getFromCache(key string) (string, error) {
	if s.redis == nil {
		return "", fmt.Errorf("redis not available")
	}
	return s.redis.Get(context.Background(), key).Result()
}

func (s *UserService) setCache(key, value string, ttl time.Duration) {
	if s.redis == nil {
		return
	}
	if err := s.redis.Set(context.Background(), key, value, ttl).Err(); err != nil {
		slog.Error("failed to set cache", "key", key, "error", err)
	}
}

func (s *UserService) delCache(key string) {
	if s.redis == nil {
		return
	}
	s.redis.Del(context.Background(), key)
}
