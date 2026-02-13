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
	fiberRecover "github.com/gofiber/fiber/v3/middleware/recover"

	"movie-discovery-api-gateway/internal/config"
	"movie-discovery-api-gateway/internal/handler"
	"movie-discovery-api-gateway/internal/middleware"
	"movie-discovery-api-gateway/internal/proxy"
)

func main() {
slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

cfg, err := config.Load()
if err != nil {
slog.Error("failed to load config", "error", err)
os.Exit(1)
}

// Connect to Redis for rate limiting
rdb, err := middleware.NewRedisClient(cfg.Redis)
if err != nil {
slog.Error("failed to connect to Redis", "error", err)
os.Exit(1)
}

// Load swagger spec
swaggerYAML, err := os.ReadFile("docs/swagger.yaml")
if err != nil {
slog.Warn("swagger spec not found, swagger UI will be unavailable", "error", err)
}

// Create Fiber app
app := fiber.New(fiber.Config{
AppName:      "api-gateway",
ServerHeader: "api-gateway",
})

// Global middleware
app.Use(fiberRecover.New())
app.Use(logger.New())
app.Use(cors.New())

// Rate limiting
rateLimiter := middleware.NewRateLimiter(rdb, cfg.RateLimitMax, cfg.RateLimitWindowSeconds)
app.Use(rateLimiter.Handler())

// Authentication (mock)
app.Use(middleware.AuthMiddleware())

// Swagger (public, bypasses auth)
if swaggerYAML != nil {
handler.RegisterSwagger(app, swaggerYAML)
}

// Health check (gateway itself)
app.Get("/health", func(c fiber.Ctx) error {
return c.JSON(fiber.Map{
"status":  "ok",
"service": "api-gateway",
})
})

// Service proxy
svcProxy := proxy.NewServiceProxy()

// Route: Movies -> Movie Service
app.All("/api/v1/movies/*", svcProxy.ForwardTo(cfg.MovieServiceURL, ""))
app.All("/api/v1/movies", svcProxy.ForwardTo(cfg.MovieServiceURL, ""))

// Route: Admin sync -> Movie Service
app.All("/api/v1/admin/*", svcProxy.ForwardTo(cfg.MovieServiceURL, ""))
app.All("/api/v1/admin/sync", svcProxy.ForwardTo(cfg.MovieServiceURL, ""))

// Route: Users & Preferences -> User Preference Service
app.All("/api/v1/users/:id/preferences", svcProxy.ForwardTo(cfg.UserPreferenceServiceURL, ""))
app.All("/api/v1/users/:id/interactions", svcProxy.ForwardTo(cfg.UserPreferenceServiceURL, ""))
app.All("/api/v1/users/:id/recommendations", svcProxy.ForwardTo(cfg.RecommendationServiceURL, ""))
app.All("/api/v1/users/*", svcProxy.ForwardTo(cfg.UserPreferenceServiceURL, ""))
app.All("/api/v1/users", svcProxy.ForwardTo(cfg.UserPreferenceServiceURL, ""))

// Route: Rules -> Recommendation Service
app.All("/api/v1/rules", svcProxy.ForwardTo(cfg.RecommendationServiceURL, ""))

// Graceful shutdown
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

go func() {
slog.Info("api-gateway starting", "port", cfg.Port)
if err := app.Listen(":" + cfg.Port); err != nil {
slog.Error("server error", "error", err)
}
}()

<-ctx.Done()
slog.Info("shutting down api-gateway...")

// Shutdown HTTP server first (stop accepting new requests)
if err := app.Shutdown(); err != nil {
slog.Error("error shutting down HTTP server", "error", err)
}
slog.Info("HTTP server stopped")

// Close Redis connection
if err := rdb.Close(); err != nil {
slog.Error("error closing Redis connection", "error", err)
} else {
slog.Info("Redis connection closed")
}

slog.Info("api-gateway shutdown complete")
}
