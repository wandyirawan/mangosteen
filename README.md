# 🥭 Mangosteen

**Lightweight Keycloak alternative in Go** - MVP for small businesses & systems.

## Why Mangosteen?

Keycloak is powerful but **overwhelming** for small projects:
- Heavy resource usage
- Complex setup
- Enterprise features you don't need
- Java dependency

**Mangosteen** = Keycloak features you actually need, minus the enterprise bloat.

## What It Replaces

| Keycloak | Mangosteen |
|---------|-----------|
| WildFly/JBoss | Go binary (~15MB) |
| PostgreSQL | SQLite (embedded) |
| Infinispan | Optional Valkey/Redis |
| OAuth | Simple JWT RS256 |
| User federation | Local users only |
| Client templates | REST API |
| Realm concepts | Single app |

## For Who?

- Small to medium systems
- Microservices needing auth
- Projects that outgrew JWT but don't need Keycloak
- Teams wanting simple IAM without ops overhead
- **Headless CMS** (e.g., Granate) needing external auth provider

## Integrations

Mangosteen is designed to be used as a lightweight auth provider for other services:

### Granate CMS Integration
Granate (Rust headless CMS) uses Mangosteen for all authentication:
- JWT validation via JWKS endpoint (`/api/.well-known/jwks.json`)
- Login/register proxy to Mangosteen APIs
- User info endpoint delegation
- See: https://github.com/wandyirawan/granate

## Stack

- **Go 1.25+** - Compile to single binary
- **Fiber** - Fast HTTP framework
- **SQLite** - Embedded DB, no setup
- **sqlc** - Type-safe SQL (no ORM)
- **Argon2id** - Secure password hashing
- **JWT RS256** - Asymmetric signing with JWKS

## Structure

```
cmd/server/      # Entry point
config/        # Configuration
sql/           # Migrations & queries
internal/      # Business logic
├── auth/       # JWT, login, register, refresh
├── user/       # User CRUD + attributes
├── crown/      # Admin console (PicoCSS + Alpine.js)
├── health/     # Health checks
├── admin/      # Admin API operations
├── middleware/  # Auth + RBAC
└── db/        # sqlc generated
pkg/           # Shared packages
├── cache/      # Valkey/Redis
├── crypto/     # Password hashing
├── logger/     # Structured logging
├── queue/      # Upload queue
└── worker/     # Background jobs
```

**Flat folder** = idiomatic Go. No complex layer hierarchy.

## Features

- ✅ JWT RS256 with JWKS endpoint
- ✅ Refresh token rotation
- ✅ RBAC (admin/user roles)
- ✅ User attributes (key-value, like Keycloak)
- ✅ Admin console (PicoCSS + Alpine.js)
- ✅ Health checks (live/ready/metrics)
- ✅ Log upload to S3/Garage (optional)
- ✅ Graceful shutdown
- ✅ Auto-reload dev mode (air)
- ✅ Superadmin bootstrap from env

## Quick Start

```bash
# Clone & run
git clone https://github.com/wandyirawan/mangosteen.git
cd mangosteen

# Generate JWT keys
chmod +x generate-certs.sh && ./generate-certs.sh

# Copy keys to .env
cp .env.example .env
# Edit .env with JWT_PRIVATE_KEY & JWT_PUBLIC_KEY

# Build and run
make run
# or: go run cmd/server/main.go
```

## Superadmin Bootstrap

Configure `.env` to auto-create admin user on first start:

```
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=admin123
```

If admin already exists, skip. Login at `/admin/login`.

## Development

```bash
# Install air (auto-reload)
go install github.com/air-verse/air/cmd/air@latest

# Run with live reload
make dev

# Build binary only
make build
```

See `.air.toml` for config.

## Admin Console

```
GET    /admin/login         # Login page
POST   /admin/login         # Sign in
GET    /admin/logout        # Sign out
GET    /admin/users         # Users table (auth required)
GET    /admin/users/:id     # User detail + tabs (auth required)
```

**User Detail tabs:** Details (email/role/active) + Attributes (key-value CRUD).


## API Endpoints

**Public:**
```
POST /api/auth/login
POST /api/auth/register
POST /api/auth/refresh
GET  /.well-known/openid-configuration
GET  /.well-known/jwks.json
GET  /api/health/live
GET  /api/health/ready
GET  /api/health/metrics
```

**Auth required:**
```
GET    /api/users/me
PATCH  /api/users/me
DELETE /api/users/me
POST   /api/auth/logout

GET    /api/users/me/attributes
PUT    /api/users/me/attributes
DELETE /api/users/me/attributes/:key
```

**Admin only:**
```
GET    /api/users/
GET    /api/users/all
GET    /api/users/:id
PATCH  /api/users/:id
PATCH  /api/users/:id/role
POST   /api/users/:id/activate
DELETE /api/users/:id

GET    /api/users/:id/attributes
PUT    /api/users/:id/attributes
DELETE /api/users/:id/attributes/:key

GET    /api/admin/logs/stats
POST   /api/admin/logs/retry
GET    /api/admin/info
```

## Differences from Keycloak

| Feature | Keycloak | Mangosteen |
|---------|---------|-----------|
| Multi-tenant | ✅ Realms | ❌ Single app |
| User federation | ✅ LDAP/AD | ❌ Local only |
| OAuth flows | ✅ Full | ❌ Simple |
| Client adapters | ✅ Many | ❌ REST only |
| Themes | ✅ UI | ❌ API |
| Conferences | ✅ Many | ❌ None |
| HA mode | ✅ Infinispan | ❌ Single |
| Database | Any SQL | SQLite default |

## Production Ready?

Yes, for:
- Small to medium load
- Single instance
- Embedded or small external DB

No, if you need:
- Multi-tenant
- LDAP integration
- High availability
- Complex oauth flows

## License

MIT - Free to use, modify, distribute.