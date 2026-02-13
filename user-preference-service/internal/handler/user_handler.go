package handler

import (
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"movie-discovery-user-preference-service/internal/models"
	"movie-discovery-user-preference-service/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Health returns service health status.
func (h *UserHandler) Health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": "user-preference-service",
	})
}

// CreateUser creates a new user.
func (h *UserHandler) CreateUser(c fiber.Ctx) error {
	var req models.CreateUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid request body"})
	}

	user, err := h.svc.CreateUser(req)
	if err != nil {
		slog.Error("failed to create user", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

// GetUser returns a user by ID.
func (h *UserHandler) GetUser(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid user ID"})
	}

	user, err := h.svc.GetUser(id)
	if err != nil {
		if err.Error() == "user not found" {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "user not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal error"})
	}

	return c.JSON(user)
}

// SetPreference sets or updates user preferences.
func (h *UserHandler) SetPreference(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid user ID"})
	}

	var req models.SetPreferenceRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid request body"})
	}

	pref, err := h.svc.SetPreference(id, req)
	if err != nil {
		if err.Error() == "user not found" {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "user not found"})
		}
		slog.Error("failed to set preference", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to set preferences"})
	}

	return c.JSON(pref)
}

// GetPreference returns user preferences.
func (h *UserHandler) GetPreference(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid user ID"})
	}

	pref, err := h.svc.GetPreference(id)
	if err != nil {
		slog.Error("failed to get preference", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to get preferences"})
	}

	return c.JSON(pref)
}

// RecordInteraction records a user interaction with a movie.
func (h *UserHandler) RecordInteraction(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid user ID"})
	}

	var req models.CreateInteractionRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid request body"})
	}

	inter, err := h.svc.RecordInteraction(id, req)
	if err != nil {
		if err.Error() == "user not found" {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "user not found"})
		}
		slog.Error("failed to record interaction", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(inter)
}

// GetInteractions returns user interactions.
func (h *UserHandler) GetInteractions(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid user ID"})
	}

	limit := fiber.Query(c, "limit", 50)

	interactions, err := h.svc.GetInteractions(id, limit)
	if err != nil {
		slog.Error("failed to get interactions", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to get interactions"})
	}

	if interactions == nil {
		interactions = []models.UserInteraction{}
	}

	return c.JSON(fiber.Map{
		"user_id":      id,
		"interactions": interactions,
	})
}
