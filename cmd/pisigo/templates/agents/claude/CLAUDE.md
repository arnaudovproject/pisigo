# Pisigo project instructions

Website: https://pisigo.com · Author: Ventsislav Arnaudov (https://varnaudov.com)  
Module: `github.com/arnaudovproject/pisigo`

This app is built with **Pisigo**. Prefer Pisigo APIs over new HTTP frameworks or ad-hoc `net/http` routers.

## Quick facts

- Boot: `pisigo.Boot()` · Handlers: `func(*pisigo.Context) error` · Middleware via `app.Use` / `Group`
- Return `*pisigo.HTTPError` (or sentinels). Use `c.Bind` + `validate` tags for input.
- Default MW: `Recover`, `RequestID`, `Logger`, `Secure`
- CLI: `pisigo new|dev|prod|generate handler|migrate create|install agents`

## Package cheat sheet

| Need | Import |
|------|--------|
| Core HTTP | `github.com/arnaudovproject/pisigo` |
| Middleware | `.../middleware` |
| Auth / session | `.../auth`, `.../session` |
| Config | `.../config` |
| SQL | `.../db` + `.../adapters/{postgres,mysql,sqlite,mssql}` |
| Mongo | `.../adapters/mongodb` |
| KV / jobs / bus | `.../store|queue|pubsub` + `adapters/{memory,redis,nats}` |
| Migrate / health / metrics / otel / openapi | matching packages |
| WS / SSE / circuit / i18n | matching packages |
| HTTP tests | `.../testing` as `pisigotest` |

## Expected app layout

```
cmd/<app>/main.go       # wire deps, middleware, Register routes, Server
internal/http/          # handlers + DTOs + Register(app, deps)
internal/service/       # use-cases
internal/repository/    # persistence
internal/domain/        # types + ports
migrations/
```

Keep handlers thin. Pass a `Deps` struct; avoid globals.

## Do

- Use route groups with leading `/` segments (`Group("/api")`, `GET("/users")` or `GET("users")`)
- Map domain errors → HTTP errors in one place
- Respect `c.Request().Context()` (Timeout, DB, outbound calls)
- Run `go test ./...` after meaningful changes

## Don't

- Don't bypass Pisigo for routing/errors/binding unless integrating stdlib-only (metrics handler, WS upgrade)
- Don't put SQL/Redis client types in domain packages — use interfaces
- Don't invent parallel config loaders; use `pisigo/config`

Topic rules under `.claude/rules/` load with this file. Prefer updating those for path-specific guidance.
