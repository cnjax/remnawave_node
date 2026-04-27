# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Remnawave Node is a proxy node manager for the Remnawave Panel. It manages the Xray proxy engine (via gRPC), handles user/inbound configuration, collects traffic stats, and provides IP blocking. The project has two fully compatible implementations:

- **TypeScript/NestJS** (`src/`) — primary production implementation, uses supervisord to manage Xray
- **Go** (`remnanode/`) — alternative implementation, manages Xray directly via `os/exec` (no supervisord needed)

Both expose the same REST API over mutual TLS with JWT (RS256) authentication.

## Commands

### TypeScript/NestJS

```bash
npm install          # Install dependencies
npm run build        # Compile TypeScript → dist/
npm run start:dev    # Development with watch mode
npm start            # Production mode
npm run lint         # ESLint check
npm run lint:fix     # ESLint auto-fix
npm run format       # Prettier format
npm run knip         # Find unused code/exports
```

There are no automated tests for the TypeScript implementation.

### Go (`remnanode/`)

```bash
cd remnanode
make deps            # go mod download && tidy
make build           # Build to ./bin/remnanode
make build-dev       # Build with race detector
make run             # go run (no binary)
make build-linux     # Cross-compile for Linux amd64
make test            # go test -v ./...
make test-cover      # Tests + HTML coverage report
make fmt             # gofmt
make lint            # golangci-lint
make clean           # Remove bin/ and coverage files
```

Run a single Go test:
```bash
cd remnanode
go test -v -run TestNamePattern ./...
go test -v -run TestName ./path/to/package
```

### Versioning & Release

```bash
make bump-patch      # Bump patch version
make bump-minor      # Bump minor version
make tag-release     # Create signed git tag & push (triggers CI)
```

## Architecture

### Request Flow

```
Control Panel → HTTPS REST API (mTLS + JWT)
                    │
            ┌───────┴────────┐
            │ Handler Module │  → Xray gRPC (HandlerService): add/remove users
            │ Stats Module   │  → Xray gRPC (StatsService): traffic metrics
            │ Xray Module    │  → Supervisord (NestJS) or os/exec (Go): lifecycle
            │ Vision Module  │  → Xray gRPC (RoutingService): IP blocking
            └───────┬────────┘
                    │
            Internal API :61001 (localhost only, no JWT)
```

### Internal Service Ports

| Port  | Service                    |
|-------|----------------------------|
| 61000 | Xray gRPC (XTLS SDK)       |
| 61001 | Internal API (vision, config fetch) |
| 61002 | Supervisord HTTP (NestJS only)      |

### API Route Prefixes

- `/node/handler/*` — user management (add/remove users, inbound queries)
- `/node/stats/*` — traffic and system statistics
- `/node/xray/*` — Xray lifecycle (start/stop/status/healthcheck)
- Internal `:61001/block-ip`, `/unblock-ip`, `/internal/get-config`

### TypeScript Module Layout (`src/modules/`)

Each module follows NestJS conventions: `*.module.ts`, `*.service.ts`, `*.controller.ts`, `models/`.

- **xray-core** — Wraps `@remnawave/xtls-sdk-nestjs` (gRPC) + `@remnawave/supervisord-nestjs`
- **handler** — Validates and forwards user config changes to Xray
- **stats** — Polls Xray StatsService; aggregates system info via `systeminformation`
- **vision** — Manages IP block/unblock rules via Xray RoutingService
- **internal** — Serves config to Xray process over localhost

### State Management (Go)

The Go implementation maintains an in-memory `state.Manager` (RWMutex-protected) holding:
- Current Xray JSON config
- Per-inbound `HashedSet` (xxhash-based O(1) user tracking)
- Empty-config hash for change detection

### Key Shared Library

`libs/contract/` publishes `@remnawave/node-contract` — the TypeScript API constants and Zod schemas consumed by both this node and the control panel.

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `NODE_PORT` | Yes | — | HTTPS server port |
| `SECRET_KEY` | Yes | — | Base64 JSON: CA cert, JWT public key, node cert+key |
| `XTLS_IP` | No | `127.0.0.1` | Xray gRPC host |
| `XTLS_PORT` | No | `61000` | Xray gRPC port |
| `DISABLE_HASHED_SET_CHECK` | No | `false` | Skip config hash comparison |
| `NODE_ENV` | No | `production` | `development` or `production` |

The `SECRET_KEY` is a base64-encoded JSON object with fields: `caCertPem`, `jwtPublicKey`, `nodeCertPem`, `nodeKeyPem`.

## Development Setup

For local development inside Docker (see `DEV_ENV.md`):

```bash
docker compose -f docker-compose-dev.yml up -d
docker exec -it remnawave-node-dev /bin/bash
# Then install NVM, Node 22, supervisord, and the Xray binary
```

Xray binary installation:
```bash
curl -L https://raw.githubusercontent.com/remnawave/scripts/main/scripts/install-latest-xray.sh -o install-xray.sh \
    && chmod +x install-xray.sh && bash ./install-xray.sh && rm install-xray.sh
```

## CI/CD

- **Dev builds**: triggered on push to `dev` branch → `remnawave/node:dev`
- **Production releases**: triggered by signed git tags → `remnawave/node:latest` + versioned tags on Docker Hub and GHCR
- Releases are created via `make tag-release` which calls `git tag -s`
