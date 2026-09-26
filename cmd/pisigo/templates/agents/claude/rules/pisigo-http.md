---
paths:
  - "**/*handler*.go"
  - "**/*route*.go"
  - "**/http/**/*.go"
  - "**/middleware/**/*.go"
---

# Pisigo HTTP

https://pisigo.com

Handlers return `error`. Bind with `c.Bind(&dto)` (`json` + `validate` tags). Respond with `c.JSON` / `String` / `NoContent` / `Redirect`.

Auth: `auth.JWT`, `auth.APIKey`, `auth.RequireRoles`. Sessions: `session.Middleware` + `session.FromContext`.

Middleware order: Recover → RequestID → Logger → Secure/CORS → auth → handler.
`Timeout` returns 504 without waiting for handlers that ignore context cancel. It does not buffer responses — once headers/body are committed, Timeout cannot replace them with 504.
`Secure` HSTS only when `c.IsHTTPS()` (TLS or trusted proxy). Context is pooled — detach before sharing across goroutines.
Always check `app.Server` / `ServerWithConfig` errors.

Errors: `pisigo.ErrNotFound`, `ErrUnauthorized`, `NewHTTPError(code, msg).WithDetails(...)`.
