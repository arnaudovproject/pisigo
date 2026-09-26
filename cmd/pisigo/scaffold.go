// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func NewProject(args []string) {
	if len(args) < 1 {
		fatal("usage: pisigo new <name>")
	}
	name := sanitizeName(args[0])
	dir := name
	if err := os.MkdirAll(filepath.Join(dir, "migrations"), 0o755); err != nil {
		fatal("create project: %v", err)
	}

	files := map[string]string{
		"go.mod": fmt.Sprintf("module %s\n\ngo 1.26.1\n\nrequire github.com/arnaudovproject/pisigo v0.0.0\n", name),
		"main.go": `package main

import (
	"log"
	"log/slog"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/health"
	"github.com/arnaudovproject/pisigo/middleware"
)

func main() {
	app := pisigo.Boot()
	app.SetLogger(pisigo.NewTextLogger(slog.LevelInfo))
	app.Use(
		middleware.Recover(),
		middleware.RequestID(),
		middleware.Logger(),
		middleware.Secure(),
	)

	health.New().Register(app, "/health")

	app.GET("/", func(c *pisigo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	if err := app.Server("0.0.0.0", 8080); err != nil {
		log.Fatal(err)
	}
}
`,
		".env.example": "PORT=8080\nLOG_LEVEL=info\n",
	}

	for path, content := range files {
		full := filepath.Join(dir, path)
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			fatal("write %s: %v", path, err)
		}
	}
	fmt.Printf("pisigo: created project %s\n", dir)
	fmt.Printf("pisigo: tip: cd %s && pisigo install agents all\n", dir)
}

func Generate(args []string) {
	if len(args) < 2 {
		fatal("usage: pisigo generate handler <name>")
	}
	kind := args[0]
	name := sanitizeName(args[1])
	switch kind {
	case "handler":
		path := name + "_handler.go"
		content := fmt.Sprintf(`package main

import "github.com/arnaudovproject/pisigo"

func Register%s(app *pisigo.App) {
	g := app.Group("/%s")
	g.GET("", list%s)
	g.POST("", create%s)
	g.GET("/{id}", get%s)
}

func list%s(c *pisigo.Context) error {
	return c.JSON(200, []any{})
}

func create%s(c *pisigo.Context) error {
	return c.JSON(201, map[string]string{"status": "created"})
}

func get%s(c *pisigo.Context) error {
	return c.JSON(200, map[string]string{"id": c.Param("id")})
}
`, exportName(name), name, exportName(name), exportName(name), exportName(name), exportName(name), exportName(name), exportName(name))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			fatal("write handler: %v", err)
		}
		fmt.Printf("pisigo: generated %s\n", path)
	default:
		fatal("unknown generate kind: %s", kind)
	}
}

func MigrateCmd(args []string) {
	if len(args) < 1 {
		fatal("usage: pisigo migrate create <name>")
	}
	switch args[0] {
	case "create":
		if len(args) < 2 {
			fatal("usage: pisigo migrate create <name>")
		}
		dir := "migrations"
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fatal("%v", err)
		}
		path, err := createMigrationFile(dir, args[1])
		if err != nil {
			fatal("%v", err)
		}
		fmt.Printf("pisigo: created %s\n", path)
	default:
		fatal("unknown migrate command: %s", args[0])
	}
}

func createMigrationFile(dir, name string) (string, error) {
	return writeMigration(dir, name)
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

func exportName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-'
	})
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}
