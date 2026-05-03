# Remnanode Install Guide

This guide covers installing the Go implementation of Remnawave Node without supervisord.

`remnanode` is the long-running service. `rw-core` is started and stopped by `remnanode` only after the panel sends `/node/xray/start` or `/node/xray/stop`.

## Build

Build a Linux amd64 binary:

```sh
cd remnanode
make build-linux
```

The output is:

```text
remnanode/bin/remnanode-linux-amd64
```

Install it on the server:

```sh
install -m 755 remnanode-linux-amd64 /usr/local/bin/remnanode
```

`rw-core` must also exist and be executable:

```sh
install -m 755 xray /usr/local/bin/rw-core
```

## Environment

Required:

```sh
NODE_PORT=50415
SECRET_KEY='...'
```

Recommended:

```sh
XRAY_CORE_VERSION=26.3.27
XRAY_BINARY_PATH=/usr/local/bin/rw-core
XTLS_API_PORT=61000
```

Notes:
- `SECRET_KEY` is the encoded node payload from the panel.
- `XTLS_API_PORT` defaults to `61000`.
- `XRAY_BINARY_PATH` defaults to `/usr/local/bin/rw-core`.
- `remnanode` listens on `NODE_PORT` with mTLS and on `127.0.0.1:61001` for the internal config API.

## systemd

The default service template is:

```text
remnanode/contrib/systemd/remnanode.service
```

Install:

```sh
install -m 755 /usr/local/bin/remnanode /usr/local/bin/remnanode
mkdir -p /etc/remnanode
install -m 600 remnanode/contrib/remnanode.env.example /etc/remnanode/remnanode.env
install -m 644 remnanode/contrib/systemd/remnanode.service /etc/systemd/system/remnanode.service
systemctl daemon-reload
systemctl enable --now remnanode
```

Check:

```sh
systemctl status remnanode
journalctl -u remnanode -f
```

## OpenRC

The default service template is:

```text
remnanode/contrib/openrc/remnanode
```

Install:

```sh
install -m 755 /usr/local/bin/remnanode /usr/local/bin/remnanode
install -m 755 remnanode/contrib/openrc/remnanode /etc/init.d/remnanode
install -m 600 remnanode/contrib/remnanode.env.example /etc/conf.d/remnanode
rc-update add remnanode default
rc-service remnanode start
```

Check:

```sh
rc-service remnanode status
tail -f /var/log/remnanode.log /var/log/remnanode.err.log
```

## Migrating Legacy OpenRC Service Name `node`

Older deployments used service name `node` and binary path `/root/remnanode`. New deployments should use service name `remnanode` and binary path `/usr/local/bin/remnanode`.

Migration:

```sh
stamp=$(date +%Y%m%d%H%M%S)

cp -a /etc/init.d/node /etc/init.d/node.bak.$stamp
cp -a /etc/conf.d/node /etc/conf.d/node.bak.$stamp
cp -a /root/remnanode /root/remnanode.bak.$stamp

install -m 755 /root/remnanode /usr/local/bin/remnanode
install -m 755 remnanode/contrib/openrc/remnanode /etc/init.d/remnanode
cp -a /etc/conf.d/node /etc/conf.d/remnanode

rc-service node stop
rc-update del node default
rc-update add remnanode default
rc-service remnanode start
```

After migration, use `rc-service remnanode ...` and `/etc/conf.d/remnanode`.

## Runtime Behavior

Starting `remnanode` does not start `rw-core`.

Startup sequence:
1. `remnanode` starts the main HTTPS API.
2. `remnanode` starts the internal config API on `127.0.0.1:61001`.
3. The panel sends `/node/xray/start` with the xray config.
4. `remnanode` stores the generated config in memory.
5. `remnanode` starts `rw-core`.
6. `rw-core` reads config from `http://127.0.0.1:61001/internal/get-config`.

This ordering is intentional. If `rw-core` starts before the panel provides config, it receives `{}` and exits.

## Failure Diagnostics

`remnanode` logs failed API requests with bounded request/response body prefixes. This is enabled in production and helps correlate service errors with the exact API call.

Failure logs include:

```text
method
path
query
status
latency
ip
user_agent
request_body
request_body_truncated
response_body
response_body_truncated
```

Only failed calls are logged:
- HTTP status `>= 400`
- command responses such as `isStarted=false`, `isStopped=false`, or `success=false`
- responses with non-null JSON `error` fields

Body capture is limited to 8 KiB. Common sensitive JSON fields such as `secret`, `token`, `password`, `uuid`, `vlessUuid`, `trojanPassword`, `ssPassword`, `id`, and `key` are redacted before logging.

Example:

```text
WRN api request failed method=GET path=/node/stats/get-system-stats status=500 response_body="{\"timestamp\":\"...\",\"path\":\"/node/stats/get-system-stats\",\"message\":\"Failed to get system stats\",\"errorCode\":\"A009\"}"
```

## Rollback

Keep a backup before replacing binaries or init scripts:

```sh
cp -a /usr/local/bin/remnanode /usr/local/bin/remnanode.bak.$(date +%Y%m%d%H%M%S)
cp -a /etc/init.d/remnanode /etc/init.d/remnanode.bak.$(date +%Y%m%d%H%M%S)
```

For an older legacy `node` service:

```sh
cp -a /root/remnanode /root/remnanode.bak.$(date +%Y%m%d%H%M%S)
cp -a /etc/init.d/node /etc/init.d/node.bak.$(date +%Y%m%d%H%M%S)
```
