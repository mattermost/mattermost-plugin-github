# Mattermost Plugin for GitHub — Cloud Agent Instructions

> **WARNING**: Do NOT commit AGENTS.md. It is generated at runtime from this file by the update script.

## Overview

This is the Mattermost Plugin for GitHub (`github`). It has a Go server component (`server/`) and a React/TypeScript webapp (`webapp/`).

## Services

| Service | Port | Container Name | Credentials |
|---------|------|----------------|-------------|
| Mattermost Server | 8065 | `mattermost-server` | admin / Admin1234! |
| PostgreSQL | 5432 | `mattermost-postgres` | mmuser / mostest |

## Environment Variables (must be set before `make deploy`)

```bash
export NVM_DIR=""
export PATH="/usr/local/go/bin:/usr/local/bin:$HOME/.local/bin:$PATH"
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD='Admin1234!'
export MM_SERVICESETTINGS_ENABLEDEVELOPER=true
```

## Build & Deploy

```bash
make deploy   # Builds server + webapp, bundles, and uploads to Mattermost
```

## Lint

```bash
cd webapp && npm run lint        # ESLint
cd webapp && npm run check-types # TypeScript type checking
make install-go-tools            # Install golangci-lint, gotestsum
make check-style                 # Full lint (Go + JS)
```

## Test

```bash
cd webapp && npm run test        # Jest tests
go test ./...                    # Go unit tests
make test                        # Both (uses gotestsum)
```

## Gotchas

- **Node.js version**: Must match `.nvmrc` (24.13.1). The update script uses `n` and disables nvm via `NVM_DIR=""`.
- **PATH**: Always prepend `/usr/local/go/bin:/usr/local/bin` to PATH and unset `NVM_DIR` before running commands.
- **Docker nested containers**: The VM uses fuse-overlayfs storage driver and iptables-legacy. Socket permissions must be set with `sudo chmod 666 /var/run/docker.sock` after Docker starts.
- **Container resumption**: Use `docker start mattermost-server` (not `docker run`) on subsequent sessions to preserve DB state and installed plugins.
- **`make deploy`**: Requires `MM_SERVICESETTINGS_ENABLEDEVELOPER=true` to build only the current-platform binary (faster).
- **npm install**: Run in `webapp/` directory. The `webapp/node_modules` directory is git-ignored.
- **AUTOMATICPREPACKAGEDPLUGINS=false**: Prevents bundled plugins from overwriting your deployed plugin on container restart.
