# Claude Code — Project Instructions

**Read [AGENTS.md](AGENTS.md) before doing any work.** It contains the full architecture reference, code conventions, and guidelines for extending this system.

## Quick Reference

- **Stack:** Go 1.23 + Gin + PostgreSQL + goqu + golang-migrate
- **Architecture:** Handler → Service → Repository (per domain module)
- **Auth:** JWT (local) + Discord OAuth2. RBAC: user < moderator < admin
- **Entry point:** `main.go` → config → DB → DI container → router → server
- **API spec:** `docs/openapi.yaml` (OpenAPI 3.1, keep in English)
- **Migrations:** `migrations/` — sequential numbered SQL files, always create up + down

## Rules

- Follow existing patterns. Every new domain goes in `internal/{domain}/` with handler, service, repository.
- Wire new handlers through `internal/di/container.go`.
- Register new routes in `internal/routing/routes.go`.
- Use `goqu` for queries, `repository.WithTransaction()` for multi-step DB ops.
- Error responses: `gin.H{"error": "message", "details": "optional"}`.
- Keep `docs/openapi.yaml` updated and in English when adding/changing endpoints.
- Use table-driven tests with `testify/assert`.
