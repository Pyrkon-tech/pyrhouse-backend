# PyrHouse Backend — Agent Guide

> This file is the primary reference for AI assistants working on this codebase.
> Read it fully before making any changes.

## Project Overview

**PyrHouse** is a warehouse inventory management system built in Go. It tracks serialized assets (by serial/PYR code), non-serialized stock items (by quantity), storage locations, and transfers between locations. It includes user management with role-based access control and supports both local (password) and Discord OAuth2 authentication.

- **Language:** Go 1.23
- **Framework:** Gin Gonic
- **Database:** PostgreSQL 15
- **Module path:** `warehouse`
- **Deployment:** DigitalOcean App Platform (Docker)

## Directory Structure

```
├── main.go                      # Entry point: config → DB → DI → router → server
├── go.mod / go.sum              # Dependencies
├── Dockerfile                   # Multi-stage Docker build
├── docker-compose.yml           # Local dev: PostgreSQL + app
├── Makefile                     # Build targets
├── start.sh                     # Container startup script
├── .env / .env.dev              # Environment variables (local)
├── docs/
│   └── openapi.yaml             # OpenAPI 3.1 specification
├── migrations/                  # Sequential SQL migrations (golang-migrate)
│   ├── 000001_init_tables.up.sql
│   ├── 000001_init_tables.down.sql
│   └── ...                      # ~57 migration files
└── internal/                    # All application code (private)
    ├── config/                  # Config loading from env vars
    ├── di/                      # Manual DI container (wires all dependencies)
    ├── database/                # PostgreSQL connection + migration runner
    ├── middleware/               # Recovery, timeout, health check, CORS
    ├── routing/                 # Route registration (public / protected / utility)
    ├── security/                # JWT auth, Discord OAuth handler, RBAC middleware
    ├── oauth/                   # Discord OAuth client (token exchange, user fetch)
    ├── models/                  # Shared data models (User, AuditLog)
    ├── repository/              # Base repository + goqu query builder + transactions
    ├── roles/                   # Role enum + hierarchy (user < moderator < admin)
    ├── errors/                  # Custom DB error types (unique violation, FK violation)
    ├── logging/                 # Zap logger initialization
    ├── rate_limiter/            # Per-IP rate limiter (memory-based)
    ├── metadata/                # PYR code generation, origin validation, asset status
    ├── auditlog/                # Audit trail logging (Auditable interface)
    ├── inventory/               # Core business domain
    │   ├── assets/              # Serialized assets (handler, service, repository)
    │   ├── stocks/              # Non-serialized stock (handler, service, repository)
    │   ├── items/               # Combined item queries
    │   ├── category/            # Item categories (auto PYR ID generation)
    │   ├── transfers/           # Location transfers (create, confirm, cancel)
    │   └── inventory_log/       # Inventory change logging
    ├── locations/               # Location CRUD
    ├── users/                   # User CRUD + points system
    ├── integrations/
    │   ├── googlesheets/        # Google Sheets API integration
    └── service_desk/            # Support ticket handling
```

## Architecture

### Three-Tier Pattern

Every domain module follows **Handler → Service → Repository**:

```
HTTP Request
  → Handler (validation, JSON binding, response formatting)
    → Service (business logic, rules, calculations)
      → Repository (SQL queries via goqu, persistence)
```

### DI Container

`internal/di/container.go` — manual dependency injection. All handlers, services, and repositories are wired in `NewAppContainer(db, cfg)`. Optional integrations (Google Sheets, Discord) degrade gracefully if config is missing.

### Middleware Chain (applied in order)

1. `RecoveryMiddleware` — panic recovery with stack trace logging
2. `TimeoutMiddleware` — request timeout (configurable via `REQUEST_TIMEOUT_SECONDS`)
3. `CORS` — gin-contrib/cors with configurable origins
4. **Per-route:** `JWTMiddleware` on protected routes, `Authorize(role)` for RBAC

### Routing

Defined in `internal/routing/routes.go`, split into three groups:

- **Public:** `/auth`, `/auth/discord`, `/auth/discord/callback`, `/users/register`, `/health`
- **Protected:** Everything else — requires JWT in `Authorization: Bearer <token>` header
- **Utility:** `/health` endpoint

### Database

- **Driver:** `lib/pq` (raw SQL) + `goqu` (query builder)
- **Migrations:** `golang-migrate/migrate` with numbered SQL files in `migrations/`
- **Transactions:** `repository.WithTransaction(db, func(tx) error)` — auto-rollback on panic
- **Connection pool:** configurable via env vars (defaults: 25 open, 5 idle)

### Authentication

Two auth providers:

1. **Local:** POST `/auth` with username/password → bcrypt verify → JWT
2. **Discord OAuth2:** GET `/auth/discord` → Discord → callback → find/create user → JWT → redirect to frontend with `?token=`

JWT claims: `{ userID, role, username, exp }`

### RBAC

Three roles with hierarchy: `user < moderator < admin`

```go
security.Authorize("admin")           // middleware — rejects if role < admin
security.IsAllowed(c, "moderator")    // runtime check
security.IsOwnerOrAllowed(c, userID, "admin")  // owner bypass
```

## Code Conventions

### Naming

| Element | Convention | Example |
|---------|-----------|---------|
| Files | `snake_case.go` | `asset_handler.go` |
| Exported types/funcs | `PascalCase` | `NewRepository`, `GetUser` |
| Private funcs | `camelCase` | `findOrCreateUser` |
| Constants | `PascalCase` | `AssetStatusInStock` |
| DB tags | `db:"column_name"` | `db:"discord_id"` |
| JSON tags | `json:"field_name"` | `json:"discord_id,omitempty"` |

### Error Handling

Always return explicit errors. Never panic in handlers.

```go
if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{
        "error":   "User-facing message",
        "details": err.Error(),  // optional, for debugging
    })
    return
}
```

DB errors are wrapped via `errors.WrapDBError()` which maps PostgreSQL error codes (23505 → UniqueViolationError, 23503 → ForeignKeyViolationError).

### Response Format

```go
// Success
c.JSON(http.StatusOK, gin.H{"data": object})
c.JSON(http.StatusOK, gin.H{"message": "Success"})
c.JSON(http.StatusCreated, object)

// Error
c.JSON(http.StatusBadRequest, gin.H{"error": "msg", "details": "optional"})
```

### Models

- Use pointers for nullable DB fields: `PasswordHash *string`
- Use `json:"-"` to hide sensitive fields (password hash)
- Use `json:"field,omitempty"` for optional fields
- Flat records from JOINs get a `TransformTo*()` method to build nested structs

### Request Validation

Use Gin's `binding` tags:
```go
type CreateUserRequest struct {
    Username string `json:"username" binding:"required,alphanum"`
    Password string `json:"password" binding:"required"`
}
```

## Key Business Models

### Asset (serialized item)
Fields: `id`, `serial`, `pyrcode` (auto-generated: `PYR-{CAT}{ID}`), `location`, `category`, `status` (in_stock/in_transit/delivered), `origin`

### StockItem (non-serialized)
Fields: `id`, `category`, `location`, `quantity`, `origin`, `status`

### Transfer
Moves assets and stock between locations. Lifecycle: `in_transit → completed` or `in_transit → cancelled`. Supports partial item removal (restore to location).

### User
Fields: `id`, `username`, `fullname`, `role`, `points`, `active`, `discord_id`, `discord_username`, `avatar_url`, `auth_provider` (local/discord)

### ItemCategory
Fields: `id`, `name` (auto-normalized), `label`, `pyr_id` (3-char auto-generated), `type` (asset/stock)

## Environment Variables

### Required
| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `JWT_SECRET` | Secret key for JWT signing |

### Optional (with defaults)
| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `REQUEST_TIMEOUT_SECONDS` | `0` (disabled) | Request timeout |
| `APP_VERSION` | `1.0.0` | Version shown in /health |
| `DB_MAX_OPEN_CONNS` | `25` | Connection pool max |
| `DB_MAX_IDLE_CONNS` | `5` | Idle connections |
| `JWT_EXPIRATION_HOURS` | `120` | Token lifetime |
| `CORS_ALLOWED_ORIGINS` | `localhost:3000,localhost:5000` | Comma-separated origins |
| `DISCORD_CLIENT_ID` | — | Discord OAuth app ID |
| `DISCORD_CLIENT_SECRET` | — | Discord OAuth secret |
| `DISCORD_REDIRECT_URI` | — | Backend callback URL |
| `FRONTEND_URL` | — | Frontend URL for OAuth redirects |

## Adding New Features — Guidelines

### Adding a new domain module

1. Create directory under `internal/` (e.g., `internal/events/`)
2. Create three files following the pattern:
   - `handler.go` — HTTP handlers, request validation, response formatting
   - `service.go` — business logic
   - `repository.go` — database queries using goqu
3. Add a handler field to `di/container.go` and wire it in `NewAppContainer()`
4. Register routes in `internal/routing/routes.go` under the appropriate group (public/protected)
5. If needed, add a migration in `migrations/` with the next sequential number

### Adding a new endpoint to existing module

1. Add handler method to the module's `handler.go`
2. Register route in `internal/routing/routes.go`
3. **⚠️ MANDATORY:** Update `docs/openapi.yaml` with:
   - Tag (if new module)
   - Path with HTTP method
   - Parameters (path, query, body)
   - Request schema (if applicable)
   - Response schemas (success + error cases)
   - Security requirements (bearerAuth if protected)

### Adding a new migration

```bash
# Create migration files (use next number in sequence)
touch migrations/000058_description.up.sql
touch migrations/000058_description.down.sql
```

The `.up.sql` should contain the forward migration, `.down.sql` the rollback. Migrations run automatically on startup or manually with `go run main.go -migrate`.

### Adding an integration

1. Create directory under `internal/integrations/`
2. Initialize in `di/container.go` with graceful degradation (log warning if config missing, don't crash)
3. Add config fields to `internal/config/config.go`

## Testing

Uses `github.com/stretchr/testify` with table-driven tests:

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case 1", "in", "out"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := DoSomething(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

Run: `go test ./...`

## Common Pitfalls

- **OpenAPI Docs:** ⚠️ **CRITICAL** — Always update `docs/openapi.yaml` when adding or modifying endpoints. Add tag, path, parameters, request/response schemas. Keep in English.
- **CORS:** Production frontend domain must be in `CORS_ALLOWED_ORIGINS` env var on DigitalOcean
- **Discord OAuth:** `DISCORD_REDIRECT_URI` must match exactly what's registered in Discord Developer Portal
- **FRONTEND_URL:** Must include protocol (`https://pyrhouse.space`, not `pyrhouse.space`)
- **New users via Discord:** Created with `active=false` — need admin activation
- **Migrations:** Always create both `.up.sql` and `.down.sql` files
- **goqu:** Use `goqu.Ex{}` for WHERE conditions, not raw strings
- **Transactions:** Always use `repository.WithTransaction()` for multi-step DB operations
- **Error package:** Use `apperrors` package (not `custom_error`) for DB error wrapping
- **Language:** All code, comments, and error messages must be in English
