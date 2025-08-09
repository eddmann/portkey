Here is the updated technical product development guide using the product name Portkey — a secure, developer-friendly, Go-based tunnel service. The guide is broken into clear, structured iterations designed for a team to build the product collaboratively, with each step having specific goals, deliverables, and tech decisions.

⸻

🪄 Portkey — Technical Product Development Guide

A secure, extensible, Go-based tunnel service for developers.

⸻

🧭 Product Vision

Portkey allows developers to securely expose their local development servers to the internet using encrypted tunnels, with full visibility, token-based access, and modern deployment tooling. Inspired by tools like LocalTunnel, Ngrok, and Expose — Portkey is designed to be:
• Written in Go (for performance, portability)
• Self-hostable with Docker/Fargate support
• Secured with token-based authentication
• Enhanced with a real-time Web UI
• Extensible with modern tunneling tech (QUIC, gRPC, etc.)

⸻

🔁 Iteration Plan

Each iteration below is designed to be independently achievable by a team. Epics, goals, deliverables, and ownership hints are included.

⸻

✅ Iteration 1 – Core Tunneling MVP

Goal: Build the basic tunnel architecture between portkey-client and portkey-server using WebSocket and subdomains.

Team Focus: Core Networking, Routing

Features
• portkey-client:
• CLI to expose a local port
• Flags: --port, --server, --subdomain
• Persistent WebSocket connection
• portkey-server:
• Accept WebSocket connections from clients
• Map subdomain → client
• HTTP handler routes requests to correct tunnel

Deliverables
• Functional tunnel between client and server
• Requests to subdomain.portkey.dev reach localhost:PORT
• Minimal Docker support for server and client

Tech
• Go net/http
• Gorilla WebSocket
• Basic subdomain registry
• Dockerfile

⸻

🔐 Iteration 2 – Token-Based Authentication & Authorization

Goal: Secure tunnel creation and limit access using tokens and subdomain scoping.

Team Focus: Auth, Token Handling

Features
• Static auth.yaml file:

tokens:

- token: abc123
  subdomains: ["project1"]
  role: user
- token: admin456
  subdomains: ["*"]
  role: admin

  • Validate tokens on tunnel registration
  • Enforce subdomain restrictions
  • Role-based API access

Deliverables
• Secure token enforcement in CLI and server
• Token-injection via --auth-token
• Middleware for auth/role enforcement

Tech
• Go config parsing (yaml.v3)
• Middleware for auth
• Role-based struct for tokens

⸻

🌐 Iteration 3 – Embedded Caddy for TLS and Proxy

Goal: Enable HTTPS out of the box using embedded Caddy.

Team Focus: Integration, DevOps

Features
• Use embedded Caddy v2 inside portkey-server
• Auto TLS with Let’s Encrypt
• Serve:
• \*.portkey.dev → tunnel proxy
• /ui, /api → Web UI + API
• Serve static UI files via Caddy

Deliverables
• Server binary with embedded Caddy
• HTTPS works automatically
• CLI flags: --https, --domain, --caddy-email

Tech
• Caddy Go module
• Internal route adapters
• Let’s Encrypt integration

⸻

📊 Iteration 4 – Web UI + Real-Time Request Logging

Goal: Visualize tunnel traffic and enable debugging for developers.

Team Focus: UI/UX, Observability, Backend API

Features
• In-memory log store for HTTP requests
• WebSocket /api/ws for live logs
• REST APIs:
• GET /api/requests
• GET /api/tunnels
• GET /api/requests/:id
• UI SPA (React/Vue)
• Live stream
• History viewer
• Request detail view

Deliverables
• Web UI served at /ui
• Token-protected UI (admin role)
• CLI flag: --enable-web-ui, --log-store=memory

Tech
• Go REST API + Gorilla WebSocket
• React/Vue + Tailwind
• Circular buffer log storage

⸻

☁️ Iteration 5 – Docker & AWS Deployments

Goal: Simplify deployment with Docker and Fargate/EC2 readiness.

Team Focus: DevOps, Infrastructure

Features
• Hardened Dockerfiles for portkey-server and portkey-client
• ECS Fargate task definition
• EC2 systemd/Docker install docs
• Docker Compose for local dev

Deliverables
• Public Docker images
• Sample ECS/EC2 deployment templates
• ECR/Fargate setup guide

Tech
• Docker, AWS CLI
• Terraform or CloudFormation templates
• Systemd service for EC2

⸻

📂 Iteration 6 – Persistent Request Logging & Replay

Goal: Store and replay past requests, helpful for debugging webhooks.

Team Focus: Storage, Developer Experience

Features
• SQLite-backed log store
• Replay HTTP requests to active tunnels
• Optional: download logs as JSON
• CLI flag: --log-store=sqlite

Deliverables
• Persistent logs across restarts
• UI “Replay” button
• Export logs (UI + API)

Tech
• SQLite (go-sqlite3 or modernc/sqlite)
• Log retention config
• Simple replay handler

⸻

🔒 Iteration 7 – OAuth & SSO Support (Enterprise)

Goal: Enable production authentication integrations

Team Focus: Identity, Security

Features
• OAuth2 via GitHub/Google
• JWT token support for API
• Per-user subdomain ownership
• CLI: --auth-mode=jwt

Deliverables
• Login via GitHub
• JWT validation middleware
• Dashboard shows user-owned tunnels

Tech
• Go-OAuth2, OIDC
• JWT parsing (golang-jwt/jwt)
• User session middleware

⸻

📁 Project Structure

/cmd
/server - portkey-server main
/client - portkey-client main
/internal
/tunnel - registry, handlers
/proxy - HTTP/WebSocket proxy logic
/auth - token and role handling
/logstore - memory/sqlite logging
/webui - APIs and static server
/caddy - embedded TLS server
/public
/index.html - UI entrypoint
/app.js
/styles.css

⸻

🚀 Build & Deploy

CLI

go build -o portkey-client ./cmd/client

Server

go build -o portkey-server ./cmd/server

Docker

docker build -t portkey/server -f Dockerfile .
docker build -t portkey/client -f Dockerfile.client .

⸻

🧱 Tech Stack Summary

Area Tech
Language Go (1.22+)
Web Framework net/http + Gorilla WS
TLS / Proxy Embedded Caddy
UI React or Vue + Tailwind
Storage Memory / SQLite
Auth Static tokens / OAuth2
Deployment Docker, ECS, EC2

⸻

✅ Final Notes

Portkey aims to be:
• Simple to run
• Secure by default
• Useful for webhooks, previews, CI/CD, and demos
• Extensible for future needs (QUIC, mTLS, OAuth, analytics)

⸻

Would you like this guide as a markdown README scaffold, GitHub project board, or broken into a Notion-style dev plan? I can generate those next.
