# 🥭 Mangosee

Go port of [Durian](https://github.com/wandyirawan/durian) - Modern IAM server.

## Stack
- **Go 1.22+**
- **Fiber** (web framework)
- **Gorm + SQLite** (database)
- **Valkey/Redis** (cache, optional with fallback)
- **Viper** (configuration)
- **Argon2id** (password hashing)
- **JWT RS256** (asymmetric signing)

## Structure
```
cmd/server/         # Entry point
internal/           # Business logic
├── auth/           # Authentication (JWT, login, register, refresh)
├── user/           # User management (CRUD, admin ops)
├── health/         # Health checks & metrics
└── middleware/     # JWT validation & RBAC
pkg/                # Shared packages
├── database/       # SQLite connection
├── cache/          # Valkey client with fallback
└── crypto/         # Argon2id password hashing
config/             # Viper configuration
```

## Quick Start

```bash
# 1. Run the generator
chmod +x create-mangosteen.sh
./create-mangosteen.sh

# 2. Generate RSA keys
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

# 3. Convert keys for .env (single line with \n)
awk 'NF {sub(/\r/, ""); printf "%s\\n",$0}' private.pem
awk 'NF {sub(/\r/, ""); printf "%s\\n",$0}' public.pem

# 4. Copy .env.example to .env and fill in JWT keys
cp .env.example .env

# 5. Install deps and run
go mod tidy
go run cmd/server/main.go
```

## Features (from Durian)
- [ ] JWT RS256 with JWKS
- [ ] Refresh token rotation
- [ ] RBAC (admin/user)
- [ ] Session audit
- [ ] Health checks (live/ready/metrics)
- [ ] Graceful shutdown
