package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"movie-discovery-recommendation-service/internal/service"
)

type RecommendationHandler struct {
	svc *service.RecommendationService
}

func NewRecommendationHandler(svc *service.RecommendationService) *RecommendationHandler {
	return &RecommendationHandler{svc: svc}
}

// Health godoc
// GET /health
func (h *RecommendationHandler) Health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": "recommendation-service",
	})
}

// GetRecommendations godoc
// GET /api/v1/users/:id/recommendations
func (h *RecommendationHandler) GetRecommendations(c fiber.Ctx) error {
	userID := fiber.Params[int](c, "id")
	if userID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user ID",
		})
	}

	limit := fiber.Query(c, "limit", 10)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	resp, err := h.svc.GetRecommendations(c.Context(), userID, limit)
	if err != nil {
		slog.Error("failed to generate recommendations", "user_id", userID, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate recommendations",
		})
	}

	return c.JSON(resp)
}

// GetRules godoc
// GET /api/v1/rules
func (h *RecommendationHandler) GetRules(c fiber.Ctx) error {
	rules, err := h.svc.GetRules(c.Context())
	if err != nil {
		slog.Error("failed to fetch rules", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch recommendation rules",
		})
	}

	return c.JSON(fiber.Map{
		"rules": rules,
	})
}
