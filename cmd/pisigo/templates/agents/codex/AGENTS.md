# AGENTS.md — Pisigo

Framework: **Pisigo** (https://pisigo.com) by Ventsislav Arnaudov (https://varnaudov.com).  
Go module: `github.com/arnaudovproject/pisigo`.

## Mission for agents

Build and change this codebase using Pisigo primitives. Minimize new dependencies for HTTP, auth, validation, health, and messaging when Pisigo already covers them.

## Bootstrapping pattern

```go
app := pisigo.Boot()
app.Use(middleware.Recover(), middleware.RequestID(), middleware.Logger(), middleware.Secure())
// register routes...
app.Server("0.0.0.0", cfg.Int("PORT", 8080))
```

Handler signature: `func(c *pisigo.Context) error`.  
Input: `c.Bind(&in)` with `json`/`validate` tags.  
Output: `c.JSON`, `c.String`, `c.NoContent`, …  
Errors: return `pisigo.Err*` or `pisigo.NewHTTPError`.

## Where code should live

```
cmd/<app>/main.go        # DI + middleware + Server
internal/http/           # routes, handlers, DTOs
internal/service/        # use-cases
internal/repository/     # SQL/Mongo/Redis implementations
internal/domain/         # types + interfaces (ports)
migrations/              # SQL files for pisigo/migrate
```

Register routes from `internal/http` via `Register(app *pisigo.App, deps Deps)`.

## Package index

- Core: `pisigo` (App, Context, Router, Server, errors, validate)
- `middleware` — Recover, Logger, RequestID, CORS, Timeout, Secure, BasicAuth, RateLimit, Compress, CSRF, Cache
- `auth` — JWT Issue/Parse/JWT MW, APIKey, RequireRoles
- `session` — cookie sessions + MemoryStore
- `config` — LoadEnvFile / LoadJSONFile / Get Int Bool Duration
- `db` + `adapters/postgres|mysql|sqlite|mssql` — SQL
- `adapters/mongodb` — MongoDB
- `store` / `queue` / `pubsub` + `adapters/memory|redis|nats`
- `migrate`, `health`, `metrics`, `otelx`, `openapi`
- `websocket`, `sse`, `circuit`, `i18n`
- Tests: `pisigo/testing` (`pisigotest.GET/POST/Do`)

## Commands

```bash
go test ./...
pisigo dev .
pisigo generate handler users
pisigo migrate create add_users
pisigo install agents all   # refresh AI agent rules
```

## Hard rules

1. Prefer interfaces at service boundaries; put Redis/NATS/SQL types in adapters/repository.
2. Middleware order: Recover first; Timeout requires context-aware handlers.
3. Do not call `mux.Handler` patterns that skip PathValue — use Pisigo routing APIs only.
4. Keep this file and agent rules accurate when conventions change.
5. English for user-facing docs and commit messages intended for shared repos.
