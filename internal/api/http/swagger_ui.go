package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const swaggerUIHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Clawflux Swagger UI</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
    <style>
      body { margin: 0; background: #fafafa; }
      .topbar { display: none; }
    </style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
      window.ui = SwaggerUIBundle({
        url: '/swagger/swagger.json',
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis],
      });
    </script>
  </body>
</html>
`

func (r *Router) handleSwaggerUI(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/swagger/" && req.URL.Path != "/swagger" {
		http.NotFound(w, req)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerUIHTML))
}

func (r *Router) handleSwaggerAssets(w http.ResponseWriter, req *http.Request) {
	name := strings.TrimPrefix(req.URL.Path, "/swagger/")
	switch name {
	case "swagger.json", "swagger.yaml":
	default:
		http.NotFound(w, req)
		return
	}

	path := filepath.Join("docs", "swagger", name)
	if _, err := os.Stat(path); err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{
			Code:    "not_found",
			Message: "swagger asset not found; run `go generate ./cmd/api` first",
		})
		return
	}
	http.ServeFile(w, req, path)
}
