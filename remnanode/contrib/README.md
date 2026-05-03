# Contrib

This directory contains init templates for running `remnanode` itself.

`rw-core` is not managed by systemd or OpenRC directly. It is started by `remnanode` after the panel sends a start/restart command with a valid xray config.

## Defaults

The templates use the same default service identity where possible:

| Item | Default |
|---|---|
| Service name | `remnanode` |
| Binary | `/usr/local/bin/remnanode` |
| Xray binary | `/usr/local/bin/rw-core` |
| Node API env | `NODE_PORT`, `SECRET_KEY` |
| Xray API env | `XTLS_API_PORT`, `XRAY_BINARY_PATH`, `XRAY_CORE_VERSION` |

## Environment Files

systemd and OpenRC use different native conventions:

| Init system | Environment file |
|---|---|
| systemd | `/etc/remnanode/remnanode.env` |
| OpenRC | `/etc/conf.d/remnanode` |

The variable names are the same. The file path is intentionally idiomatic for each init system.

Older OpenRC deployments may use service name `node` and `/etc/conf.d/node`. Migrate those to `remnanode` and `/etc/conf.d/remnanode` when possible.
