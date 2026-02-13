package handler

import (
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"movie-discovery-movie-service/internal/models"
	"movie-discovery-movie-service/internal/service"
)

// MovieHandler handles HTTP requests for movies.
type MovieHandler struct {
	svc *service.MovieService
}

// NewMovieHandler creates a new MovieHandler.
func NewMovieHandler(svc *service.MovieService) *MovieHandler {
	return &MovieHandler{svc: svc}
}

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error string `json:"error"`
}

// Health returns service health status.
// @Summary Health check
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *MovieHandler) Health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": "movie-service",
	})
}

// ListMovies returns a paginated list of movies.
// @Summary List movies
// @Tags movies
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Param sort_by query string false "Sort field" Enums(release_date,title,popularity) default(popularity)
// @Param order query string false "Sort order" Enums(asc,desc) default(desc)
// @Param release_date_from query string false "Filter start date (YYYY-MM-DD)"
// @Param release_date_to query string false "Filter end date (YYYY-MM-DD)"
// @Success 200 {object} models.MovieListResponse
// @Failure 500 {object} ErrorResponse
// @Router /movies [get]
func (h *MovieHandler) ListMovies(c fiber.Ctx) error {
	params := models.MovieListParams{
		Page:            fiber.Query(c, "page", 1),
		PageSize:        fiber.Query(c, "page_size", 20),
		SortBy:          c.Query("sort_by", "popularity"),
		Order:           c.Query("order", "desc"),
		ReleaseDateFrom: c.Query("release_date_from"),
		ReleaseDateTo:   c.Query("release_date_to"),
	}

	result, err := h.svc.ListMovies(params)
	if err != nil {
		slog.Error("failed to list movies", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "failed to retrieve movies",
		})
	}

	return c.JSON(result)
}

// GetMovieDetail returns detailed info for a single movie.
// @Summary Get movie detail
// @Tags movies
// @Produce json
// @Param id path int true "Movie ID"
// @Success 200 {object} models.MovieDetail
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /movies/{id} [get]
func (h *MovieHandler) GetMovieDetail(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "invalid movie ID",
		})
	}

	detail, err := h.svc.GetMovieDetail(id)
	if err != nil {
		if err.Error() == "movie not found" {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "movie not found",
			})
		}
		slog.Error("failed to get movie detail", "id", id, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "failed to retrieve movie details",
		})
	}

	return c.JSON(detail)
}

// SyncMovies triggers a sync of movies from TMDB.
// @Summary Sync movies from TMDB
// @Tags admin
// @Produce json
// @Param pages query int false "Number of pages to sync" default(5)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /admin/sync [post]
func (h *MovieHandler) SyncMovies(c fiber.Ctx) error {
	pages := fiber.Query(c, "pages", 5)
	if pages < 1 {
		pages = 1
	}
	if pages > 50 {
		pages = 50
	}

	count, err := h.svc.SyncMovies(pages)
	if err != nil {
		slog.Error("sync failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "sync failed: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":       "sync completed",
		"movies_synced": count,
		"pages":         pages,
	})
}
