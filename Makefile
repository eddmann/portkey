# Portkey Makefile

BIN_DIR := bin
SERVER_BIN := $(BIN_DIR)/portkey-server
CLIENT_BIN := $(BIN_DIR)/portkey-cli

.PHONY: all build build-server build-client run-server run-client dummy-server test clean

all: build

# ---------- Build ----------

build: build-server build-client

build-server:
	@echo "Building server…"
	@mkdir -p $(BIN_DIR)
	go build -o $(SERVER_BIN) ./cmd/server

build-client:
	@echo "Building client…"
	@mkdir -p $(BIN_DIR)
	go build -o $(CLIENT_BIN) ./cmd/client

# ---------- Run ----------

run-server: build-server
	@echo "Starting portkey-server on :8080"
	@$(SERVER_BIN) -addr :8080

run-client: build-client
	@echo "Starting portkey-cli forwarding localhost:3000 as myapp"
	@$(CLIENT_BIN) --server http://localhost:8080 --subdomain myapp --port 3000

# Dummy local HTTP server that replies with "pong"
dummy-server:
	@echo "Starting dummy HTTP server on :3000 (responds with 'pong')"
	@node -e "require('http').createServer((req,res)=>{res.writeHead(200,{ 'Content-Type':'text/plain'}); res.end('pong');}).listen(3000,()=>console.log('Dummy server listening on http://localhost:3000'));"

# ---------- Tests ----------

test:
	go test ./...

# ---------- Clean ----------

clean:
	rm -rf $(BIN_DIR)
