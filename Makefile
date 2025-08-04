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

# Dummy local HTTP server that replies with \"pong\"
.ONESHELL:
dummy-server:
	@echo "Starting dummy HTTP server on :3000 (responds with 'pong')"
	python3 - <<'PY'
	import http.server, socketserver, sys
	PORT = 3000
	class Handler(http.server.SimpleHTTPRequestHandler):
	    def do_GET(self):
	        self.send_response(200)
	        self.send_header('Content-type', 'text/plain')
	        self.end_headers()
	        self.wfile.write(b'pong')
	with socketserver.TCPServer(('', PORT), Handler) as httpd:
	    print(f"Serving dummy app at http://localhost:{PORT}")
	    try:
	        httpd.serve_forever()
	    except KeyboardInterrupt:
	        sys.exit(0)
PY

# ---------- Tests ----------

test:
	go test ./...

# ---------- Clean ----------

clean:
	rm -rf $(BIN_DIR)
