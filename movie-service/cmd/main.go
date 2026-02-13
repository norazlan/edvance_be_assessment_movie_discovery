package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"movie-discovery-movie-service/internal/config"
	"movie-discovery-movie-service/internal/database"
	"movie-discovery-movie-service/internal/handler"
	"movie-discovery-movie-service/internal/repository"
	"movie-discovery-movie-service/internal/service"
	"movie-discovery-movie-service/internal/tmdb"
)

func main() {
	// Structured logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Connect to PostgreSQL
	db, err := database.NewPostgres(cfg.DB)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}

	// Connect to Redis (non-fatal if unavailable)
	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		slog.Warn("Redis unavailable, running without cache", "error", err)
	}

	// Initialize TMDB client
	tmdbClient := tmdb.NewClient(cfg.TMDB.APIKey, cfg.TMDB.BaseURL)

	// Initialize layers
	repo := repository.NewMovieRepository(db)
	svc := service.NewMovieService(repo, tmdbClient, rdb)
	h := handler.NewMovieHandler(svc)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Movie Service",
		ServerHeader: "Movie-Service",
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			slog.Error("unhandled error", "error", err, "status", code)
			return c.Status(code).JSON(handler.ErrorResponse{Error: err.Error()})
		},
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	// Swagger docs
	swaggerYAML, err := os.ReadFile("docs/swagger.yaml")
	if err != nil {
		slog.Warn("swagger.yaml not found, swagger UI will be unavailable", "error", err)
	} else {
		handler.RegisterSwagger(app, swaggerYAML)
	}

	// API routes
	api := app.Group("/api/v1")
	api.Get("/health", h.Health)
	api.Get("/movies", h.ListMovies)
	api.Get("/movies/:id", h.GetMovieDetail)
	api.Post("/admin/sync", h.SyncMovies)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server
	go func() {
		addr := ":" + cfg.Port
		slog.Info("starting movie service", "addr", addr)
		if err := app.Listen(addr); err != nil {
			slog.Error("server error", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down movie service...")

	// Shutdown HTTP server first (stop accepting new requests)
	if err := app.Shutdown(); err != nil {
		slog.Error("error shutting down HTTP server", "error", err)
	}
	slog.Info("HTTP server stopped")

	// Close database connections
	if err := db.Close(); err != nil {
		slog.Error("error closing PostgreSQL connection", "error", err)
	} else {
		slog.Info("PostgreSQL connection closed")
	}

	if rdb != nil {
		if err := rdb.Close(); err != nil {
			slog.Error("error closing Redis connection", "error", err)
		} else {
			slog.Info("Redis connection closed")
		}
	}

	slog.Info("movie service shutdown complete")
}
