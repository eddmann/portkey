<p align="center">
  <img src="logo.png" alt="Portkey Logo" width="300"/>
</p>

# Portkey

_Not all tunnels are in Gringotts._

> ⚠️ **Warning**: Portkey is early-stage experimental software. Review the code and run in a safe environment before using in production.

Portkey lets developers expose a local port to the internet through an encrypted tunnel – ideal for web-hooks, previews, and live demos.

## Why Portkey?

Many teams rely on SaaS tunnelling services (Ngrok, LocalTunnel, Cloud-flared etc.) during local development. In our case we needed:

- Secure web-hook testing and preview links during development & QA **inside a private AWS VPC**.
- Complete control of TLS certificates and traffic logs (security & compliance).
- A tiny, self-contained client that could run **as a Docker image or standalone binary**, easy to vendor into any pipeline.

Portkey was created as a lightweight, self-hostable alternative that works the same on a laptop, inside Docker-Compose, or on any cloud provider.
At the moment Portkey handles **HTTP/HTTPS tunnels**; future versions will add additional transports (gRPC, WebSockets over QUIC, TCP streams).
It is **self-hostable**, written in Go, ships with an embedded Caddy HTTPS proxy, and features a real-time Web UI.

---

## What’s Inside

| Area        | Feature                                                                                        |
| ----------- | ---------------------------------------------------------------------------------------------- |
| Core Tunnel | bidirectional WebSocket tunnel (`portkey-client ↔ portkey-server`)                             |
| Auth        | Static token auth with wildcard sub-domain rules (`auth.yaml`)                                 |
| HTTPS       | Embedded Caddy v2 – automatic Let’s Encrypt (`--use-caddy`)                                    |
| Logging     | In-memory log buffer + optional SQLite persistence (`--log-store=sqlite`, `--log-retention=N`) |
| Web UI      | Vanilla-JS SPA at `/ui` – live stream, search, pagination, dark-mode                           |
| Admin APIs  | `/api/requests`, `/api/tunnels`, `/api/ws` (admin-token gated)                                 |
| Docker      | Scratch images (`portkey/server`, `portkey/client`) + `docker-compose.yml` stack               |

---

## Quick Start (Local)

```bash
# build binaries
make build

# start dummy local service
autodummy(){ python3 -m http.server 3000; }; autodummy &

# run server (HTTP 8080, Web UI enabled)
./bin/portkey-server -addr :8080 \
  --auth-file auth.yaml          \
  --enable-web-ui

# expose the dummy service
./bin/portkey-client --server http://localhost:8080 \
  --subdomain myapp              \
  --host localhost --port 3000   \
  --auth-token admin456
```

Visit `http://localhost:8080/ui` and use token `admin456` to watch live requests.

---

## Docker / Compose

```bash
# build & spin up the full stack (server+client+dummy)
make compose-up
```

The stack persists logs to `./data/portkey.db` (SQLite).

---

## Server Flags

| Flag              | Default | Description                                            |
| ----------------- | ------- | ------------------------------------------------------ |
| `--auth-file`     |         | Path to `auth.yaml`; if omitted, server runs open.     |
| `--use-caddy`     | false   | Enable embedded Caddy HTTPS reverse-proxy.             |
| `--caddy-domain`  |         | Domain for certificates (required with `--use-caddy`). |
| `--enable-web-ui` | false   | Serve `/ui` and admin APIs.                            |
| `--log-store`     | memory  | `memory` or `sqlite` log backend.                      |
| `--log-db`        | logs.db | SQLite filename when `--log-store=sqlite`.             |
| `--log-retention` | 0       | Purge logs older than N days (SQLite only).            |

## Client Flags

| Flag           | Default   | Description                          |
| -------------- | --------- | ------------------------------------ |
| `--host`       | localhost | Local hostname of service to expose. |
| `--port`       | 3000      | Local port to expose.                |
| `--auth-token` |           | Token to authenticate with server.   |

---

## Tests & Admin APIs

### Admin APIs (token=admin)

| Endpoint                | Description                            |
| ----------------------- | -------------------------------------- |
| `GET /api/requests`     | JSON array of recent or persisted logs |
| `GET /api/requests/:id` | Single log entry                       |
| `GET /api/tunnels`      | Active sub-domains                     |

### Running tests

```bash
go test ./integration   # e2e suites
go test ./...           # all
```

---

Contributions & feedback are welcome – open an issue or pull request!
