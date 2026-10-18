package handler

import (
	_ "embed"

	"github.com/gofiber/fiber/v3"
)

// openapiSpec is the API description in OpenAPI 3 format. go:embed puts
// the file inside the compiled program, so the server needs no extra
// files at run time.
//
//go:embed openapi.yaml
var openapiSpec []byte

// swaggerPage shows the description with Swagger UI. The UI itself is
// loaded from a CDN with a fixed version, so we do not store its files.
const swaggerPage = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Swap Platform API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({ url: '/api/docs/openapi.yaml', dom_id: '#swagger-ui' });
  </script>
</body>
</html>`

// DocsHandler serves the API documentation.
type DocsHandler struct{}

// NewDocsHandler creates the handler.
func NewDocsHandler() *DocsHandler { return &DocsHandler{} }

// UI handles GET /api/docs: the Swagger page.
func (h *DocsHandler) UI(c fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return c.SendString(swaggerPage)
}

// Spec handles GET /api/docs/openapi.yaml: the description itself.
func (h *DocsHandler) Spec(c fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, "application/yaml; charset=utf-8")
	return c.Send(openapiSpec)
}
