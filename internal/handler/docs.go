package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gfc-app-finance/greencard-mobile/exchange/docs"
)

func RegisterDocsRoutes(router *gin.Engine) {
	router.GET("/docs/openapi.yaml", func(c *gin.Context) {
		data, err := docs.OpenAPISpec.ReadFile("openapi.yaml")
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to load spec")
			return
		}
		c.Data(http.StatusOK, "application/x-yaml", data)
	})

	router.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIHTML))
	})
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Exchange API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #fafafa; }
    #swagger-ui .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/docs/openapi.yaml",
      dom_id: "#swagger-ui",
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: "BaseLayout",
      deepLinking: true,
    });
  </script>
</body>
</html>`
