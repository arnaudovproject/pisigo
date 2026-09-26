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

Errors: `pisigo.ErrNotFound`, `ErrUnauthorized`, `NewHTTPError(code, msg).WithDetails(...)`.
