package handler

import (
	"github.com/gofiber/fiber/v3"
)

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>User Preference Service API - Swagger UI</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        body { margin: 0; padding: 0; }
        #swagger-ui { max-width: 1200px; margin: 0 auto; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({
            url: '/swagger/doc.yaml',
            dom_id: '#swagger-ui',
            presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
            layout: 'StandaloneLayout'
        });
    </script>
</body>
</html>`

func RegisterSwagger(app fiber.Router, yamlContent []byte) {
	app.Get("/swagger/doc.yaml", func(c fiber.Ctx) error {
		c.Set("Content-Type", "application/yaml")
		return c.Send(yamlContent)
	})

	app.Get("/swagger/*", func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return c.SendString(swaggerHTML)
	})
}
