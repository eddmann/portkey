# Portkey Makefile

SHELL := /bin/bash

BIN_DIR := bin
SERVER_BIN := $(BIN_DIR)/portkey-server
CLIENT_BIN := $(BIN_DIR)/portkey-client

# Optimised build flags
GOFLAGS := -trimpath -buildvcs=false -ldflags="-s -w"

.PHONY: all build build-server build-client docker-build docker-push run-server run-server-ui run-client compose-up dummy-server test clean

all: build

# ---------- Build ----------

build: build-server build-client

build-server:
	@echo "Building server…"
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build $(GOFLAGS) -tags "netgo osusergo" -o $(SERVER_BIN) ./cmd/server

build-client:
	@echo "Building client…"
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build $(GOFLAGS) -tags "netgo osusergo" -o $(CLIENT_BIN) ./cmd/client

# ---------- Run ----------

run-server: build-server
	@echo "Starting portkey-server on :8080"
	@$(SERVER_BIN) --port 8080 -auth-file integration/auth.yaml

run-server-ui: build-server
	@echo "Starting portkey-server with Web UI on :8080"
	@$(SERVER_BIN) --port 8080 -auth-file integration/auth.yaml --enable-web-ui

docker-build:
	@docker build -t portkey/server -f Dockerfile.server .
	@docker build -t portkey/client -f Dockerfile.client .

docker-push:
	@docker push portkey/server
	@docker push portkey/client

compose-up:
	@docker compose up --build

run-client: build-client
	@echo "Starting portkey-client forwarding localhost:3000 as myapp"
	@$(CLIENT_BIN) --server http://localhost:8080 --subdomain myapp --host localhost --port 3000 --auth-token admin456

# Dummy local HTTP server that replies with "pong"
dummy-server:
	@echo "Starting enhanced dummy HTTP server on :3000 (echoes request headers)"
	@node -e "require('http').createServer((req,res)=>{console.log('--- Incoming request:', req.method, req.url); console.log(req.headers); const responseHeaders={ 'Content-Type':'application/json', 'X-Dummy':'1' }; res.writeHead(200,responseHeaders); res.end(JSON.stringify({ message:'pong', receivedHeaders:req.headers },null,2));}).listen(3000,()=>console.log('Dummy server listening on http://localhost:3000'));"

# ---------- Tests ----------

test:
	go test ./...

# ---------- Clean ----------

clean:
	rm -rf $(BIN_DIR)
