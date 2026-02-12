package proxy

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

// ServiceProxy forwards requests to downstream microservices.
type ServiceProxy struct {
	client *http.Client
}

// NewServiceProxy creates a new service proxy with sensible defaults.
func NewServiceProxy() *ServiceProxy {
	return &ServiceProxy{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// ForwardTo creates a handler that proxies requests to the given baseURL.
// The pathPrefix is stripped before forwarding.
func (p *ServiceProxy) ForwardTo(baseURL, pathPrefix string) fiber.Handler {
	baseURL = strings.TrimRight(baseURL, "/")

	return func(c fiber.Ctx) error {
		// Build target URL: strip the gateway prefix, forward the rest
		originalPath := c.Path()
		targetPath := originalPath
		if pathPrefix != "" {
			targetPath = strings.TrimPrefix(originalPath, pathPrefix)
			if targetPath == "" {
				targetPath = "/"
			}
		}

		targetURL := baseURL + targetPath
		if q := string(c.Request().URI().QueryString()); q != "" {
			targetURL += "?" + q
		}

		slog.Debug("proxying request",
			"method", c.Method(),
			"from", originalPath,
			"to", targetURL,
		)

		// Build the outgoing request
		var bodyReader io.Reader
		if len(c.Body()) > 0 {
			bodyReader = strings.NewReader(string(c.Body()))
		}

		req, err := http.NewRequestWithContext(c.Context(), c.Method(), targetURL, bodyReader)
		if err != nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
				"error": "failed to create proxy request",
			})
		}

		// Forward relevant headers
		req.Header.Set("Content-Type", c.Get("Content-Type", "application/json"))
		if auth := c.Get("Authorization"); auth != "" {
			req.Header.Set("Authorization", auth)
		}
		req.Header.Set("X-Forwarded-For", c.IP())
		req.Header.Set("X-Forwarded-Host", c.Hostname())

		// Execute the request
		resp, err := p.client.Do(req)
		if err != nil {
			slog.Error("proxy request failed", "url", targetURL, "error", err)
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
				"error": fmt.Sprintf("service unavailable: %s", baseURL),
			})
		}
		defer resp.Body.Close()

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
				"error": "failed to read service response",
			})
		}

		// Copy response headers
		for key, vals := range resp.Header {
			for _, val := range vals {
				c.Set(key, val)
			}
		}

		return c.Status(resp.StatusCode).Send(body)
	}
}
