# 🥭 Mangosteen

**Lightweight Keycloak alternative in Go** — MVP for small businesses & systems.

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
- **Salad Buah ecosystem** — IAM for Granate, Salak, Kelapa, Duwet

## Integrations

Mangosteen is the IAM backbone for the Salad Buah (Pomegranate) ecosystem:

| Service | How it uses Mangosteen |
|---------|----------------------|
| **Granate** (Rust CMS) | JWT validation via JWKS, login/register proxy |
| **Salak** (Product Service) | JWT verification on all protected endpoints |
| **Kelapa** (Ecommerce) | Admin login, service tokens for API calls |
| **Duwet** (Warehouse TUI) | Login via `/api/auth/login`, token for Salak API |

All services verify tokens by fetching JWKS from `/.well-known/jwks.json`.

## Stack

- **Go 1.25+** — Compile to single binary
- **Fiber** — Fast HTTP framework
- **SQLite** — Embedded DB, no setup
- **sqlc** — Type-safe SQL (no ORM)
- **Argon2id** — Secure password hashing
- **JWT RS256** — Asymmetric signing with JWKS endpoint

## Structure

```
cmd/server/      # Entry point
config/          # Configuration
sql/             # Migrations & queries
internal/        # Business logic
├── auth/         # JWT, login, register, refresh
├── user/         # User CRUD + attributes
├── crown/        # Admin console (PicoCSS + Alpine.js)
├── health/       # Health checks
├── admin/        # Admin API operations
├── middleware/    # Auth + RBAC
└── db/           # sqlc generated
pkg/             # Shared packages
├── cache/        # Valkey/Redis
├── crypto/       # Password hashing
├── logger/       # Structured logging
├── queue/        # Upload queue
└── worker/       # Background jobs
```

**Flat folder** = idiomatic Go. No complex layer hierarchy.

## Features

- ✅ JWT RS256 with JWKS endpoint
- ✅ RSA key management (PKCS#1 + PKCS#8)
- ✅ Key rotation support
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
# Clone
git clone https://github.com/wandyirawan/mangosteen.git
cd mangosteen

# Generate RSA key pair
mkdir -p keys
openssl genrsa -out keys/private.pem 2048
openssl rsa -in keys/private.pem -pubout -out keys/public.pem

# Configure
cp .env.example .env
# Required: JWT_PRIVATE_KEY=keys/private.pem
# Required: JWT_PUBLIC_KEY=keys/public.pem

# Run
go run cmd/server/main.go
# → http://localhost:4000
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP listen port | `4000` |
| `DATABASE_DSN` | SQLite database path | `mangosee.db` |
| `JWT_ISSUER` | JWT issuer claim | `mangosteen` |
| `JWT_ACCESS_TTL` | Access token TTL (minutes) | `15` |
| `JWT_REFRESH_TTL` | Refresh token TTL (days) | `7` |
| `JWT_PRIVATE_KEY` | Path to RSA private key PEM | — |
| `JWT_PUBLIC_KEY` | Path to RSA public key PEM | — |
| `ADMIN_EMAIL` | Bootstrap admin email | — |
| `ADMIN_PASSWORD` | Bootstrap admin password | — |

### JWT Key Setup

Mangosteen uses **RS256** (RSA 2048-bit) for JWT signing. Without keys configured, falls back to HS256 (development only).

**Key file paths** — set in `.env`:
```env
JWT_PRIVATE_KEY=keys/private.pem
JWT_PUBLIC_KEY=keys/public.pem
```

Both **PKCS#1** (`RSA PRIVATE KEY`) and **PKCS#8** (`PRIVATE KEY`) formats are supported.

The JWKS endpoint serves public keys at `/.well-known/jwks.json`:
```json
{"keys":[{"kid":"a97ecceb","kty":"RSA","alg":"RS256","use":"sig","n":"...","e":"AQAB"}]}
```

### Integrating Services

Services verify Mangosteen tokens by:
1. Fetch JWKS from `http://localhost:4000/.well-known/jwks.json`
2. Extract `kid` from JWT header
3. Find matching public key in JWKS
4. Verify signature with RS256

```python
# Python (Salak example)
import jwt, requests
from jwt.algorithms import RSAAlgorithm

jwks = requests.get("http://localhost:4000/.well-known/jwks.json").json()
header = jwt.get_unverified_header(token)
key = next(k for k in jwks["keys"] if k["kid"] == header["kid"])
public_key = RSAAlgorithm.from_jwk(key)
payload = jwt.decode(token, public_key, algorithms=["RS256"])
```

```rust
// Rust (Granate example)
let jwks: JwkSet = reqwest::get(jwks_url).await?.json().await?;
let header = decode_header(token)?;
let jwk = jwks.find(&header.kid.unwrap())?;
let key = DecodingKey::from_jwk(jwk)?;
let claims = decode::<Claims>(token, &key, &Validation::new(Algorithm::RS256))?;
```

## Superadmin Bootstrap

Configure `.env` to auto-create admin user on first start:

```
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=***
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

MIT — Free to use, modify, distribute.
