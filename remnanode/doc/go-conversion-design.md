# Remnawave Node - Go Implementation Design Document

## Overview

This document describes the Go implementation of Remnawave Node, converted from the original TypeScript/NestJS implementation. The Go version maintains full API compatibility with the TypeScript version while leveraging Go's performance characteristics.

## Architecture

### Technology Stack

| Component | TypeScript | Go |
|-----------|------------|-----|
| HTTP Framework | NestJS | Gin |
| gRPC Client | @remnawave/xtls-sdk | xtls/xray-core direct |
| Process Management | node-supervisord | Direct process control (os/exec) |
| JWT | passport-jwt (RS256) | golang-jwt/jwt/v5 |
| Logging | Winston | zerolog |
| Hashing | @remnawave/hashed-set | xxhash/v2 |
| Compression | zstd (axios) | klauspost/compress/zstd |

### Project Structure

```
remnanode/
├── cmd/remnanode/main.go     # Application entry point
├── internal/
│   ├── config/               # Configuration management
│   ├── server/               # HTTP server setup
│   │   └── middleware/       # JWT, logging, recovery, compression
│   ├── handler/              # User management module
│   ├── stats/                # Statistics module
│   ├── xray/                 # Xray lifecycle module
│   ├── vision/               # IP blocking module
│   ├── internal_api/         # Internal API module
│   ├── xray_client/          # gRPC client for Xray
│   ├── process/              # Direct process management (os/exec)
│   ├── state/                # State management
│   └── errors/               # Error codes and responses
├── pkg/
│   ├── hashedset/            # Thread-safe hashed set
│   ├── sysinfo/              # System information
│   └── response/             # Standard response format
├── go.mod
├── Makefile
└── doc/
```

## API Endpoints

All endpoints maintain the same paths and request/response formats as the TypeScript version.

### Handler Module (`/node/handler/*`)

| Method | Path | Description |
|--------|------|-------------|
| POST | /node/handler/add-user | Add a single user |
| POST | /node/handler/add-users | Add multiple users (bulk) |
| POST | /node/handler/remove-user | Remove a single user |
| POST | /node/handler/remove-users | Remove multiple users (bulk) |
| POST | /node/handler/get-inbound-users | Get users in an inbound |
| POST | /node/handler/get-inbound-users-count | Get user count in an inbound |

### Stats Module (`/node/stats/*`)

| Method | Path | Description |
|--------|------|-------------|
| GET | /node/stats/get-system-stats | Get Xray system statistics |
| POST | /node/stats/get-user-online-status | Check if user is online |
| POST | /node/stats/get-users-stats | Get all users' traffic stats |
| POST | /node/stats/get-inbound-stats | Get inbound traffic stats |
| POST | /node/stats/get-outbound-stats | Get outbound traffic stats |
| POST | /node/stats/get-all-inbounds-stats | Get all inbounds' stats |
| POST | /node/stats/get-all-outbounds-stats | Get all outbounds' stats |
| POST | /node/stats/get-combined-stats | Get combined inbound/outbound stats |

### Xray Module (`/node/xray/*`)

| Method | Path | Description |
|--------|------|-------------|
| POST | /node/xray/start | Start Xray with configuration |
| GET | /node/xray/stop | Stop Xray process |
| GET | /node/xray/status | Get Xray status and version |
| GET | /node/xray/healthcheck | Get node health status |

### Vision Module (Internal Port 61001)

| Method | Path | Description |
|--------|------|-------------|
| POST | /block-ip | Block an IP address |
| POST | /unblock-ip | Unblock an IP address |

### Internal API (Internal Port 61001)

| Method | Path | Description |
|--------|------|-------------|
| GET | /internal/get-config | Get current Xray configuration |

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| NODE_PORT | Yes | - | Main HTTPS server port |
| SECRET_KEY | Yes | - | Base64-encoded JSON with certificates |
| XTLS_IP | No | 127.0.0.1 | Xray gRPC server IP |
| XTLS_PORT | No | 61000 | Xray gRPC server port |
| DISABLE_HASHED_SET_CHECK | No | false | Disable config hash checking |
| XRAY_CORE_VERSION | No | - | Xray core version string |
| NODE_ENV | No | production | Environment (development/production) |

### SECRET_KEY Format

```json
{
  "caCertPem": "-----BEGIN CERTIFICATE-----...",
  "jwtPublicKey": "-----BEGIN PUBLIC KEY-----...",
  "nodeCertPem": "-----BEGIN CERTIFICATE-----...",
  "nodeKeyPem": "-----BEGIN PRIVATE KEY-----..."
}
```

## Security

### mTLS Configuration

The main HTTPS server requires mutual TLS:
- Server presents `nodeCertPem` signed by CA
- Clients must present certificate signed by `caCertPem`
- TLS 1.2 minimum

### JWT Authentication

- Algorithm: RS256 (RSA with SHA-256)
- Public key from SECRET_KEY payload
- Bearer token in Authorization header
- Applied to all `/node/*` endpoints

### Internal API Protection

- Bound to `127.0.0.1:61001` only
- Additional IP check middleware
- No JWT required (localhost only)

### Request Compression

The server supports compressed request bodies:
- **zstd** (Zstandard): Primary compression used by axios client
- **gzip**: Fallback compression support

Compression is handled by middleware that:
1. Checks `Content-Encoding` header for `zstd` or `gzip`
2. Auto-detects compression by magic bytes if header is missing
3. Decompresses body before passing to handlers

## State Management

### HashedSet

Thread-safe set using xxhash for 64-bit hashing:
- O(1) add/delete/has operations
- Deterministic combined hash for set comparison
- Used for tracking users in inbounds

### State Manager

Maintains:
- Current Xray configuration
- Empty config hash for change detection
- Per-inbound user hash sets
- Active inbound tags

## gRPC Communication

### Xray Core API

Connects to Xray gRPC API at `XTLS_IP:XTLS_PORT`:
- HandlerService: User management (add/remove users)
- StatsService: Traffic and system statistics
- RoutingService: IP blocking rules

### Message Size

Maximum gRPC message size: 100MB (for large configurations)

## Process Management

### Direct Process Control

The Go implementation manages the Xray process directly using `os/exec`, eliminating the need for supervisord:

```go
// Command executed:
/usr/local/bin/rw-core -config http://127.0.0.1:61001/internal/get-config -format json
```

Features:
- Direct process spawning and termination
- Stdout/stderr streaming to console via zerolog
- Graceful shutdown with SIGTERM, fallback to SIGKILL
- Process state tracking with mutex protection
- Automatic restart on configuration changes

### Health Check

- 10 retry attempts
- 2 second interval
- Uses GetSysStats gRPC call as health indicator

## Building and Running

### Build

```bash
cd remnanode
make deps
make build
```

### Run

```bash
export SECRET_KEY="base64_encoded_secret_key"
export NODE_PORT="3000"
./bin/remnanode
```

### Development

```bash
make run  # Run directly without building
make build-dev  # Build with race detector
```

## Error Codes

| Code | Description |
|------|-------------|
| A001 | Internal server error |
| A002 | Login error |
| A003 | Unauthorized |
| A004 | Forbidden role |
| A010 | Failed to get system stats |
| A011 | Failed to get users stats |
| A012 | Failed to get inbound stats |
| A013 | Failed to get outbound stats |
| A014 | Failed to get inbound users |
| A015 | Failed to get inbounds stats |
| A016 | Failed to get outbounds stats |
| A017 | Failed to get combined stats |

## Response Format

All responses follow this format:

```json
{
  "isOk": true,
  "code": "optional_error_code",
  "message": "optional_message",
  "response": { /* actual response data */ }
}
```

## Migration Notes

### Key Differences from TypeScript

1. **No dependency injection framework**: Services are created manually in main.go
2. **No decorators**: Route registration is explicit in routes.go
3. **Synchronous by default**: Go uses goroutines instead of Promises
4. **Static typing**: All types are defined at compile time
5. **Error handling**: Explicit error returns instead of exceptions
6. **Process management**: Direct os/exec instead of supervisord XML-RPC
7. **Compression**: Explicit zstd/gzip decompression middleware

### Compatibility

- All API endpoints maintain the same paths
- Request/response JSON formats are identical
- Environment variables are the same
- SECRET_KEY format unchanged
- Supervisord is no longer required (process managed directly)

## Performance Considerations

1. **Connection pooling**: gRPC connection reused across requests
2. **Mutex optimization**: RWMutex for read-heavy state operations
3. **Memory efficiency**: No garbage collection pauses from node.js
4. **Startup time**: Faster cold starts compared to NestJS

## Future Improvements

1. Add OpenTelemetry tracing
2. Implement graceful degradation for gRPC failures
3. Add Prometheus metrics endpoint
4. Support configuration hot-reload
5. Remove debug logging after stabilization
6. Add request/response compression for outgoing responses
