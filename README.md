# 🪄 Portkey – Iteration 1: Core Tunneling MVP

Portkey lets you expose a local port to the internet through a secure, token-ready tunnel.
This README documents the **Iteration 1** milestone – a minimal but functional tunnel built with Go.

---

## ✨ Features Delivered

- `portkey-server` – accepts WebSocket tunnel connections and routes HTTP traffic to them.
- `portkey-cli` – establishes a persistent WebSocket tunnel and forwards requests to your local server.
- Sub-domain registry (in-memory, concurrency-safe).
- **Token-based authentication & authorization** with YAML config.
- Black-box integration tests: one with token auth, one with auth disabled.
- Container images via multi-stage Dockerfiles.

---

## 🏎️ Quick Start

```bash
# 1. Clone & build
make build          # or: go build -o bin/portkey-server ./cmd/server
                    #       go build -o bin/portkey-cli    ./cmd/client

# 2. Prepare auth file (tokens & roles)
cat > auth.yaml <<EOF
tokens:
  - token: abc123
    subdomains: ["project1"]
    role: user
  - token: admin456
    subdomains: ["*"]
    role: admin
EOF

# 3. Start the server (listen on 8080)
./bin/portkey-server -addr :8080 --auth-file auth.yaml

# 3. Start your local app (example React dev server)
cd myapp && npm run dev           # assumes it listens on :3000

# 4. Run portkey-cli to expose it
./bin/portkey-cli --server http://localhost:8080 \
                --subdomain myapp \
                --port 3000 \
                --auth-token admin456

# 5. From another terminal / browser
curl -H "Host: myapp.localhost" http://localhost:8080/
```

You should see your local application’s response.

#### Without Authentication

If you prefer open access (for local testing), simply skip the `--auth-file` flag on the server and `--auth-token` flag on the client:

```bash
./bin/portkey-server -addr :8080              # auth disabled
./bin/portkey-cli --server http://localhost:8080 --subdomain public --port 3000
```

---

## 🧩 Directory Layout

```
cmd/
  server/   → main for portkey-server
  client/   → main for portkey-cli
internal/
  registry/ → subdomain → connection map
  tunnel/   → JSON message schemas
integration/ → black-box integration test
Dockerfile[.client] → multi-stage containers
```

---

## 🔌 How It Works

```mermaid
sequenceDiagram
    participant Browser
    participant Server
    participant CLI
    participant LocalApp

    Browser->>Server: GET https://myapp.portkey.dev/
    Server->>CLI: JSON Request (via WebSocket)
    CLI->>LocalApp: HTTP request localhost:PORT
    LocalApp-->>CLI: HTTP response
    CLI-->>Server: JSON Response (WebSocket)
    Server-->>Browser: HTTP response
```

---

## 🐳 Docker

Build images:

```bash
docker build -t portkey/server   -f Dockerfile        .
docker build -t portkey/client   -f Dockerfile.client .
```

Run:

```bash
docker run -p 8080:8080 portkey/server
```

Then start the CLI container and point it at the server container’s address.

---

## 🧪 Tests

- Unit tests: `go test ./...`
- Black-box integration test: `go test ./integration -v`

---

## 🚧 Next Iterations

1. Embedded Caddy for TLS termination.
2. Web UI for real-time request logging.

Refer to `SPEC.md` for the full roadmap.
