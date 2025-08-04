# 🪄 Portkey – Secure Tunnel Service

_Not all tunnels are in Gringotts._

Portkey lets developers expose a local port to the internet through an encrypted tunnel – ideal for web-hooks, previews, and live demos.

## ❓ Why Portkey?
Many teams rely on SaaS tunnelling services (Ngrok, LocalTunnel, Cloud-flared etc.) during local development. In our case we needed:

* Web-hook testing and preview links during CI **inside a private AWS VPC** – commercial services were blocked.
* Complete control of TLS certificates and traffic logs (security & compliance).
* A tiny, self-contained CLI that could run **as a Docker image or standalone binary**, easy to vendor into any pipeline.

Portkey was created as a lightweight, self-hostable alternative that works the same on a laptop, inside Docker-Compose, or on any cloud provider.  
It is **self-hostable**, written in Go, ships with an embedded Caddy HTTPS proxy, and features a real-time Web UI.

---

## ✨ What’s Inside (v0.2)

| Area        | Feature                                                                                        |
| ----------- | ---------------------------------------------------------------------------------------------- |
| Core Tunnel | bidirectional WebSocket tunnel (`portkey-cli ↔ portkey-server`)                                |
| Auth        | Static token auth with wildcard sub-domain rules (`auth.yaml`)                                 |
| HTTPS       | Embedded Caddy v2 – automatic Let’s Encrypt (`--use-caddy`)                                    |
| Logging     | In-memory log buffer + optional SQLite persistence (`--log-store=sqlite`, `--log-retention=N`) |
| Web UI      | Vanilla-JS SPA at `/ui` – live stream, search, pagination, dark-mode, replay placeholder       |
| Admin APIs  | `/api/requests`, `/api/tunnels`, `/api/ws` (admin-token gated)                                 |
| Docker      | Scratch images (`portkey/server`, `portkey/cli`) + `docker-compose.yml` stack                  |
| Tests       | End-to-end (e2e) suites for auth & no-auth scenarios                                           |

---

## 🏁 Quick Start (Local)

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
./bin/portkey-cli --server http://localhost:8080 \
  --subdomain myapp              \
  --host localhost --port 3000   \
  --auth-token admin456
```

Visit `http://localhost:8080/ui` and use token `admin456` to watch live requests.

---

## 🐳 Docker / Compose

```bash
# build & spin up the full stack (server+cli+dummy)
make compose-up
```

The stack persists logs to `./data/portkey.db` (SQLite).

---

## 🔌 Flags Overview

| Flag              | Default | Description                                          |
| ----------------- | ------- | ---------------------------------------------------- |
| `--auth-file`     |         | Path to `auth.yaml`; if omitted, server runs open.   |
| `--use-caddy`     | false   | Enable embedded Caddy HTTPS reverse-proxy.           |
| `--caddy-domain`  |         | Domain for certificates (required w/ `--use-caddy`). |
| `--enable-web-ui` | false   | Serve `/ui` and admin APIs.                          |
| `--log-store`     | memory  | `memory` or `sqlite` log backend.                    |
| `--log-db`        | logs.db | SQLite filename when `--log-store=sqlite`.           |
| `--log-retention` | 0       | Purge logs older than N days (SQLite only).          |

CLI additional flags:
`--host`, `--port`, `--auth-token`, `replay` sub-command (up-coming).

---

## 🧪 Tests & Admin APIs

### Admin APIs (token=admin)

| Endpoint                          | Description                            |
| --------------------------------- | -------------------------------------- |
| `GET /api/requests`               | JSON array of recent or persisted logs |
| `GET /api/requests/:id`           | Single log entry                       |
| `GET /api/tunnels`                | Active sub-domains                     |
| `GET /api/requests?download=json` | **(todo)** stream ND-JSON export       |
| `POST /api/replay/:id`            | **(todo)** replay stored request       |

### Running tests

```bash
go test ./integration   # e2e suites
go test ./...           # all
```

---

## 🛣️ Roadmap (from SPEC.md)

### Near-Term

1. **Replay Capability**  
   • `/api/replay/{id}` endpoint & `portkey-cli replay`  
   • UI “Replay” button.
2. **Request Export**  
   • `/api/requests?download=json` (ND-JSON / gz).
3. **SQLite improvements**  
   • Background vacuum / DB stats endpoint.

### Future Iterations

4. **TLS & Proxy Enhancements** – QUIC, mTLS.
5. **Web UI Dashboard** – tunnel graphs, request charts.
6. **OAuth / SSO (Enterprise)** – GitHub & Google login.
7. **Cloud Deployment** – Terraform module, AWS Fargate templates.
8. **Analytics & Usage Quotas** – optional metering plugin.

Contributions & feedback are welcome – open an issue or pull request!
