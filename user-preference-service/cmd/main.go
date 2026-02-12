package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"movie-discovery-user-preference-service/internal/config"
	"movie-discovery-user-preference-service/internal/database"
	"movie-discovery-user-preference-service/internal/handler"
	"movie-discovery-user-preference-service/internal/repository"
	"movie-discovery-user-preference-service/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := database.NewPostgres(cfg.DB)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		slog.Warn("Redis unavailable, running without cache", "error", err)
	}

	repo := repository.NewUserRepository(db)
	svc := service.NewUserService(repo, rdb)
	h := handler.NewUserHandler(svc)

	app := fiber.New(fiber.Config{
		AppName:      "User Preference Service",
		ServerHeader: "User-Preference-Service",
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(handler.ErrorResponse{Error: err.Error()})
		},
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	swaggerYAML, err := os.ReadFile("docs/swagger.yaml")
	if err != nil {
		slog.Warn("swagger.yaml not found", "error", err)
	} else {
		handler.RegisterSwagger(app, swaggerYAML)
	}

	api := app.Group("/api/v1")
	api.Get("/health", h.Health)

	// User management
	api.Post("/users", h.CreateUser)
	api.Get("/users/:id", h.GetUser)

	// Preferences
	api.Post("/users/:id/preferences", h.SetPreference)
	api.Get("/users/:id/preferences", h.GetPreference)

	// Interactions
	api.Post("/users/:id/interactions", h.RecordInteraction)
	api.Get("/users/:id/interactions", h.GetInteractions)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		slog.Info("shutting down user preference service...")
		_ = app.Shutdown()
	}()

	addr := ":" + cfg.Port
	slog.Info("starting user preference service", "addr", addr)
	if err := app.Listen(addr); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
