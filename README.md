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
internal/     # Business logic (no subdirs)
├── auth/      # JWT, login, register, refresh
├── user/      # User CRUD
├── health/    # Health checks
├── admin/     # Admin operations
├── middleware/
└── db/       # sqlc generated
pkg/          # Shared packages
└── cache/
└── worker/    # Background jobs
```

**Flat folder** = idiomatic Go. No complex layer hierarchy.

## Features

- ✅ JWT RS256 with JWKS endpoint
- ✅ Refresh token rotation
- ✅ RBAC (admin/user roles)
- ✅ Health checks (live/ready/metrics)
- ✅ Log upload to S3/Garage
- ✅ Graceful shutdown

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

# Run
go run cmd/server/main.go
```

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