---
paths:
  - "**/repository/**/*.go"
  - "**/migrations/**/*.sql"
  - "**/db/**/*.go"
  - "**/*_repository.go"
---

# Pisigo data

https://pisigo.com

SQL via `db.SQL` (`Get`/`Select`/`Exec`/`Transaction`) and adapters `postgres|mysql|sqlite|mssql`. Tests: `sqlite.OpenMemory()`.

Migrations: `pisigo migrate create <name>` then `migrate.New(db, "migrations").Up(ctx)`. Sections `-- +migrate Up/Down`.

KV/jobs: `store`/`queue`/`pubsub` interfaces; `adapters/memory` locally, `redis`/`nats` in production. Depend on interfaces in services.
