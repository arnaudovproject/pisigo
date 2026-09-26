# Pisigo structure

https://pisigo.com

- `cmd/<app>/main.go` — composition root only
- `internal/http` — Register routes + handlers + DTOs
- `internal/service` — business logic
- `internal/repository` — adapters to db/store
- `internal/domain` — entities + ports
- Feature folders over giant files; one Register per feature is fine

Named routes: `.SetName("resource.action")` + `app.Reverse`.
Static files: `app.Static` / `app.File`.
