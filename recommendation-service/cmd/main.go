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

	"movie-discovery-recommendation-service/internal/config"
	"movie-discovery-recommendation-service/internal/database"
	"movie-discovery-recommendation-service/internal/handler"
	"movie-discovery-recommendation-service/internal/repository"
	"movie-discovery-recommendation-service/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

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
	defer db.Close()

	// Connect to Redis
	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// Initialize layers
	repo := repository.NewRecommendationRepository(db)
	svc := service.NewRecommendationService(repo, rdb, cfg.MovieServiceURL, cfg.UserPreferenceServiceURL)
	h := handler.NewRecommendationHandler(svc)

	// Load swagger spec
	swaggerYAML, err := os.ReadFile("docs/swagger.yaml")
	if err != nil {
		slog.Warn("swagger spec not found, swagger UI will be unavailable", "error", err)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "recommendation-service",
		ServerHeader: "recommendation-service",
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	// Swagger
	if swaggerYAML != nil {
		handler.RegisterSwagger(app, swaggerYAML)
	}

	// Routes
	app.Get("/health", h.Health)

	api := app.Group("/api/v1")
	api.Get("/users/:id/recommendations", h.GetRecommendations)
	api.Get("/rules", h.GetRules)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("recommendation-service starting", "port", cfg.Port)
		if err := app.Listen(":" + cfg.Port); err != nil {
			slog.Error("server error", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down recommendation-service")
	_ = app.Shutdown()
}
