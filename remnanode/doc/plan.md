## Project

Go reimplementation of https://github.com/remnawave/node, maintaining full API compatibility with the TypeScript/NestJS original.

## Test server

Alpine Linux — `root@103.179.44.9 -p 50413`

| Path | Description |
|---|---|
| `/root/remnanode` | Our Go binary |
| `/usr/local/bin/rw-core` | Xray-core binary (latest release) |
| Service name | `node` (OpenRC) |

Service commands:
```sh
rc-service node start/stop/restart/status
```

## Deploy workflow

```sh
# 1. Build Linux binary
cd remnanode && make build-linux

# 2. Stop service, upload binary, start
ssh -p 50413 root@103.179.44.9 "rc-service node stop"
scp -P 50413 bin/remnanode-linux-amd64 root@103.179.44.9:/root/remnanode
ssh -p 50413 root@103.179.44.9 "chmod +x /root/remnanode && rc-service node start"
```

To update rw-core (download locally, upload — server cannot reach GitHub directly):
```sh
curl -fsSL -L https://github.com/XTLS/Xray-core/releases/download/vX.Y.Z/Xray-linux-64.zip -o /tmp/xray.zip
cd /tmp && unzip -o xray.zip xray -d /tmp/xray-out
scp -P 50413 /tmp/xray-out/xray root@103.179.44.9:/tmp/xray-new
ssh -p 50413 root@103.179.44.9 "chmod +x /tmp/xray-new && mv /tmp/xray-new /usr/local/bin/rw-core"
```

## Sync status

| Upstream tag | Our status |
|---|---|
| v2.5.0 | initial Go port base |
| v2.7.0 | ✅ synced (2026-04-27) |

### What was synced in v2.5.0 → v2.7.0

- Upgraded xray-core dependency `v1.8.24 → v1.260327.0`
- Restructured `GetSystemStats` response: `{xrayInfo, plugins, system.stats}` with `memoryUsed`, `loadAvg`
- Added user types: `shadowsocks22`, `hysteria`
- Added handler endpoints: `drop-users-connections`, `drop-ips`
- Added stats endpoints: `get-user-ip-list`, `get-users-ip-list` (xray `GetStatsOnlineIpList` gRPC)
- Config: `XTLS_API_PORT` is now canonical; `XTLS_IP` removed (always `127.0.0.1`)
- Deployed rw-core v25.12.8 → v26.3.27

## Known gaps vs TypeScript

- `drop-users-connections` / `drop-ips`: endpoints exist and return `{success:true}` but do not actually RST connections — requires `CAP_NET_ADMIN` + OS-level socket destroy (equivalent of `sockdestroy` npm package). TypeScript degrades the same way when CAP_NET_ADMIN is unavailable.
- Plugin system (torrent blocker, nftables, egress/ingress filters): not implemented. `plugins.torrentBlocker.reportsCount` is always 0.
- Network interface stats in `system.stats.interface`: always `null` (field present, not populated).
