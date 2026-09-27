package openapi

import (
	"encoding/json"
	"html"
	"net/http"

	"github.com/arnaudovproject/pisigo"
)

type Doc struct {
	OpenAPI string         `json:"openapi"`
	Info    Info           `json:"info"`
	Paths   map[string]any `json:"paths"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

func FromApp(app *pisigo.App, title, version string) Doc {
	if title == "" {
		title = "pisigo API"
	}
	if version == "" {
		version = "1.0.0"
	}
	paths := map[string]any{}
	for _, route := range app.Routes() {
		item, _ := paths[route.Path].(map[string]any)
		if item == nil {
			item = map[string]any{}
		}
		op := map[string]any{
			"responses": map[string]any{
				"200": map[string]any{"description": "OK"},
			},
		}
		if route.Name != "" {
			op["operationId"] = route.Name
		}
		item[methodKey(route.Method)] = op
		paths[route.Path] = item
	}
	return Doc{
		OpenAPI: "3.0.3",
		Info:    Info{Title: title, Version: version},
		Paths:   paths,
	}
}

func methodKey(method string) string {
	switch method {
	case http.MethodGet:
		return "get"
	case http.MethodPost:
		return "post"
	case http.MethodPut:
		return "put"
	case http.MethodPatch:
		return "patch"
	case http.MethodDelete:
		return "delete"
	case http.MethodHead:
		return "head"
	case http.MethodOptions:
		return "options"
	default:
		return "get"
	}
}

func Register(app *pisigo.App, path, title, version string) {
	if path == "" {
		path = "/openapi.json"
	}
	app.GET(path, func(c *pisigo.Context) error {
		return c.JSON(200, FromApp(app, title, version))
	})
	safeTitle := html.EscapeString(title)
	safePath, _ := json.Marshal(path)
	app.GET("/docs", func(c *pisigo.Context) error {
		htmlDoc := `<!doctype html><html><head><title>` + safeTitle + `</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head><body><div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>SwaggerUIBundle({url:` + string(safePath) + `,dom_id:'#swagger-ui'})</script>
</body></html>`
		return c.HTML(200, htmlDoc)
	})
}

func JSON(app *pisigo.App, title, version string) ([]byte, error) {
	return json.MarshalIndent(FromApp(app, title, version), "", "  ")
}
