# Pisigo

**Pisigo** is a lightweight, production-oriented HTTP framework for Go.

Website: [https://pisigo.com](https://pisigo.com)  
Author: [Ventsislav Arnaudov](https://varnaudov.com)  
Module: `github.com/arnaudovproject/pisigo`

---

## Why Pisigo

Pisigo gives you a small core (router, context, middleware, errors) and optional adapters for SQL, MongoDB, Redis, NATS, metrics, tracing, auth, sessions, and more — without forcing a heavy application architecture.

- Familiar handler style with explicit errors
- First-class middleware and route groups
- Built-in request binding and validation
- Graceful shutdown, TLS, trusted proxies
- Pluggable store / queue / pubsub interfaces
- CLI for scaffolding, local development, migrations, and AI agent rules

---

## Requirements

- Go **1.26.1+**

---

## Install

### Library

```bash
go get github.com/arnaudovproject/pisigo@latest
```

### CLI

```bash
go install github.com/arnaudovproject/pisigo/cmd/pisigo@latest
```

---

## Quick start

```go
package main

import (
	"log"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/middleware"
)

func main() {
	app := pisigo.Boot()
	app.Use(
		middleware.Recover(),
		middleware.RequestID(),
		middleware.Logger(),
		middleware.Secure(),
	)

	app.GET("/", func(c *pisigo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	if err := app.Server("0.0.0.0", 8080); err != nil {
		log.Fatal(err)
	}
}
```

### Scaffold a project

```bash
pisigo new myapp
cd myapp
# wire the module replace / go get as needed
pisigo install agents all    # optional: Cursor / Claude / Codex project rules
pisigo dev .
```

---

## AI agent rules

Pisigo can install **project-local instructions** for popular coding agents so they understand the framework (routing, handlers, middleware, auth, adapters, recommended layout) with less prompting and fewer tokens.

### Install into your app

Run from your application root (the project that already depends on Pisigo):

```bash
pisigo install agents all
pisigo install agents cursor
pisigo install agents claude
pisigo install agents codex
pisigo install agents cursor claude --force
pisigo install agents all -d ./myapp
```

| Target | What gets created | Used by |
|--------|-------------------|---------|
| `cursor` | `.cursor/rules/pisigo-*.mdc` | [Cursor](https://cursor.com) project rules |
| `claude` | `CLAUDE.md` + `.claude/rules/pisigo-*.md` | [Claude Code](https://code.claude.com/docs/en/claude-md) |
| `codex` | `AGENTS.md` | [OpenAI Codex](https://agents.md/) / AGENTS.md-compatible tools |
| `all` | Everything above | — |

Flags:

- `-d, --dir <path>` — project root (default `.`)
- `-f, --force` — overwrite existing files (default skips existing)

### What the rules contain

Dense, English guidance covering:

- How Pisigo boots, routes, groups, and handles errors
- Recommended `cmd/` + `internal/{http,service,repository,domain}` layout
- Middleware, auth, sessions, binding/validation patterns
- SQL / Mongo / store / queue / pubsub adapter usage
- Migrations, health, metrics, and CLI commands

Commit the generated files in **your application repo** so the whole team gets the same agent context. Re-run with `--force` after upgrading Pisigo if templates improve.

---

## Core concepts

### App and routing

```go
app := pisigo.Boot()
app.GET("/users/{id}", handler).SetName("user.show")
app.POST("/users", createUser)

api := app.Group("/api", authMiddleware)
api.GET("/me", me)

path, _ := app.Reverse("user.show", "id", "42") // /users/42
```

Supported methods: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`, `CONNECT`, `TRACE`, plus `ANY`.

### Context

`*pisigo.Context` wraps request/response helpers:

| Area | Helpers |
|------|---------|
| Request | `Param`, `Query`, `Bind`, `BindJSON`, `FormValue`, `BearerToken`, `Body` |
| Response | `JSON`, `HTML`, `XML`, `String`, `Data`, `File`, `Download`, `Redirect`, `NoContent` |
| Store | `Set`, `Get`, `MustGet` |
| Misc | `IP`, `IsHTTPS`, `Log`, `Cookie`, `Header` |

**Pooling:** Context values are reused from a `sync.Pool`. Do not retain or access a `*pisigo.Context` after the handler returns (including from another goroutine) unless you call `DetachFromPool()` and later `Release()`.

### Errors

Return `*pisigo.HTTPError` (or sentinels like `pisigo.ErrNotFound`). The default error handler renders JSON:

```go
return pisigo.NewHTTPError(400, "invalid input").WithDetails(map[string]any{
	"field": "email",
})
```

### Validation

`c.Bind(&in)` unmarshals JSON/XML and validates `validate` struct tags via [go-playground/validator](https://github.com/go-playground/validator).

---

## Middleware

| Middleware | Purpose |
|------------|---------|
| `Recover` | Panic recovery (stack stays in logs, client gets generic 500) |
| `Logger` | Structured request logging (no bodies/secrets by default) |
| `RequestID` | `X-Request-ID` propagation (validated; max 128 chars `[A-Za-z0-9_-]`) |
| `CORS` | Cross-origin headers (credentials-safe) |
| `Timeout` | Request timeout — returns 504 without blocking on slow handlers; handlers must respect `c.Request().Context()` |
| `Secure` | Security headers; HSTS only for TLS or trusted-proxy HTTPS |
| `BasicAuth` | HTTP Basic |
| `RateLimit` | Token / window limiting |
| `Compress` | Gzip responses |
| `CSRF` | CSRF cookie + header/form check |
| `Cache` | Response cache via `store.Store` |

```go
app.Use(
	middleware.Recover(),
	middleware.RequestID(),
	middleware.Logger(),
	middleware.CORS(),
	middleware.Timeout(15*time.Second),
)
```

---

## Auth and sessions

```go
cfg := auth.DefaultJWTConfig("your-secret")
token, _ := auth.Issue(cfg, "user-id", []string{"admin"}, nil)

app.Use(auth.JWT(cfg), auth.RequireRoles("admin"))
app.Use(auth.APIKey("X-API-Key", func(k string) bool { return k == "secret" }))

app.Use(session.Middleware(session.Config{
	Store: session.NewMemoryStore(),
	// Secure cookies are on by default. For local HTTP:
	// InsecureCookie: true,
}))
```

---

## Production checklist

Use these defaults and settings for internet-facing services:

1. **CORS** — allowlist exact origins; never combine `AllowOrigins: ["*"]` with `AllowCredentials: true` (Pisigo ignores the wildcard in that case).
2. **Trusted proxies** — call `app.SetTrustedProxies(...)` only for your load balancer / reverse proxy CIDRs. Client IP and `IsHTTPS()` / HSTS honor forwarded headers only from trusted peers.
3. **Cookies** — session and CSRF cookies are `Secure` by default; use `InsecureCookie: true` / `CSRF(CSRFConfig{Secure: false})` only on plain HTTP.
4. **WebSocket** — default `CheckOrigin` is same-origin; pass a custom allowlist via `websocket.Config` if needed.
5. **Postgres** — DSN default `sslmode=require`; set `SSLMode: "disable"` only for local databases.
6. **Body size** — default max body is 1 MiB (`SetMaxBodyBytes`); form/multipart parsing respects the same limit.
7. **Metrics** — protect the endpoint: `m.Register(app, "/metrics", middleware.BasicAuth(...))`.
8. **Realtime (SSE/WS)** — use `pisigo.StreamingServerConfig()` (WriteTimeout disabled) or set timeouts explicitly via `ServerWithConfig`.
9. **Rate limit** — store errors fail closed (503); prefer Redis rate limit store in multi-instance deployments.
10. **Timeout** — handlers must respect `c.Request().Context()`; Timeout returns 504 promptly and does not wait for ignored cancellation.
11. **Server errors** — `Server` / `ServerWithConfig` return `error` (e.g. bind failure); always check it.

```go
if err := app.ServerWithConfig("0.0.0.0", 8080, pisigo.StreamingServerConfig()); err != nil {
	log.Fatal(err)
}
```

---

## Data layer

### SQL (`db` + adapters)

```go
import "github.com/arnaudovproject/pisigo/adapters/sqlite"

sqlDB, err := sqlite.OpenMemory()
// postgres, mysql, mssql adapters available
```

`db.SQL` provides `Get`, `Select`, `Exec`, `Query`, `Transaction`, pool options, and DSN builders.

### MongoDB

```go
import "github.com/arnaudovproject/pisigo/adapters/mongodb"

mdb, err := mongodb.Open(mongodb.Config{
	URI: "mongodb://localhost:27017",
	Database: "app",
})
```

### Store / Queue / PubSub

Interfaces live in `store`, `queue`, and `pubsub`.

| Backend | Store | Queue | PubSub | Rate limit |
|---------|-------|-------|--------|------------|
| Memory | yes | yes | yes | via middleware |
| Redis | yes | yes | — | yes |
| NATS | — | JetStream | yes | — |

---

## Migrations

```bash
pisigo migrate create add_users
```

```go
m := migrate.New(sqlDB, "migrations")
_ = m.Up(ctx)
_ = m.Down(ctx, 1)
```

SQL files use:

```sql
-- +migrate Up
CREATE TABLE users (...);

-- +migrate Down
DROP TABLE users;
```

---

## Ops

### Health

```go
h := health.New()
h.Ready("db", func(ctx context.Context) error { return sqlDB.Ping(ctx) })
h.Register(app, "/health") // /health/live, /health/ready
```

### Metrics (Prometheus)

```go
m := metrics.New("myapp")
app.Use(m.Middleware())
m.Register(app, "/metrics", middleware.BasicAuth(middleware.BasicAuthConfig{
	Validator: func(u, p string) bool { return u == "prom" && p == "secret" },
}))
```

### OpenTelemetry

```go
shutdown, _ := otelx.Setup("myapp")
defer shutdown(context.Background())
app.Use(otelx.Middleware("myapp"))
```

`Setup` configures W3C TraceContext propagation. Swap the stdout exporter for OTLP in your app when shipping to a collector.

### OpenAPI

```go
openapi.Register(app, "/openapi.json", "My API", "1.0.0")
// also serves /docs (Swagger UI)
```

---

## Realtime

- **WebSocket** — `websocket.Upgrade` / `websocket.Handler`
- **SSE** — `sse.Open` + `Event` / `Comment` / `Retry`

---

## CLI

| Command | Description |
|---------|-------------|
| `pisigo new <name>` | Scaffold a new app |
| `pisigo dev [path]` | Run with auto-reload |
| `pisigo prod [path] [-o out]` | Production build |
| `pisigo generate handler <name>` | Handler stub |
| `pisigo migrate create <name>` | Migration file |
| `pisigo install agents <target>` | Install AI agent rules (`cursor`, `claude`, `codex`, or `all`) |

```bash
pisigo install agents all
pisigo install agents cursor claude --force
```

See [AI agent rules](#ai-agent-rules) for formats and details.

---

## Testing

Use the built-in helpers:

```go
import pisigotest "github.com/arnaudovproject/pisigo/testing"

res := pisigotest.GET(app, "/hello")
res = pisigotest.POST(app, "/in", map[string]string{"email": "a@b.com"})
```

Run the suite:

```bash
go test ./...
```

CI runs the same suite on every push and pull request.

---

## Project layout

```
pisigo/
├── app.go, router.go, context.go, server.go   # HTTP core
├── middleware/                                # HTTP middleware
├── auth/, session/, config/, i18n/            # Cross-cutting
├── db/, migrate/, adapters/                   # Data adapters
├── store/, queue/, pubsub/                    # Messaging interfaces
├── health/, metrics/, otelx/, openapi/        # Observability / docs
├── websocket/, sse/, circuit/                 # Realtime & resilience
├── testing/                                   # Test helpers
└── cmd/pisigo/                                # CLI (+ embedded agent templates)
```

---

## Configuration

```go
cfg, _ := config.LoadEnvFile(".env")
port := cfg.Int("PORT", 8080)
debug := cfg.Bool("DEBUG", false)
timeout := cfg.Duration("TIMEOUT", 15*time.Second)
```

Also supports `LoadJSONFile` (nested keys flattened as `db.host`) and `LoadEnv`.

---

## License and authorship

Framework site: [https://pisigo.com](https://pisigo.com)  
Author: **Ventsislav Arnaudov** — [https://varnaudov.com](https://varnaudov.com)

Every source file carries this attribution. Contributions should keep the same header.
