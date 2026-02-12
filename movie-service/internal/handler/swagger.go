package handler

import (
	"github.com/gofiber/fiber/v3"
)

// RegisterSwagger registers swagger routes on the given Fiber app.
func RegisterSwagger(app fiber.Router, yamlContent []byte) {
	app.Get("/swagger/doc.yaml", func(c fiber.Ctx) error {
		c.Set("Content-Type", "application/yaml")
		return c.Send(yamlContent)
	})

	app.Get("/swagger/*", func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Movie Service - Swagger UI</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
    <style>html{box-sizing:border-box;overflow-y:scroll}*,*:before,*:after{box-sizing:inherit}body{margin:0;background:#fafafa}</style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
    SwaggerUIBundle({
        url: "/swagger/doc.yaml",
        dom_id: "#swagger-ui",
        presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
        layout: "BaseLayout"
    });
    </script>
</body>
</html>`
		return c.SendString(html)
	})
}
