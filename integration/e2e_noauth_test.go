package integration

import (
	"context"
	"fmt"
	"io"

	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"

	"strings"
	"testing"
	"time"
)

func TestBlackboxNoAuth(t *testing.T) {
    tempDir := t.TempDir()

    serverBin := filepath.Join(tempDir, "portkey-server")
    clientBin := filepath.Join(tempDir, "portkey-client")

    buildBinary(t, "../cmd/server", serverBin)
    buildBinary(t, "../cmd/client", clientBin)

    // local dummy app
    local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("pong")) }))
    defer local.Close()
    localPort := strings.Split(local.URL, ":")[2]

    srvPort, _ := findFreePort()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // server without auth-file
    srvCmd := exec.CommandContext(ctx, serverBin, "--port", fmt.Sprintf("%d", srvPort))
    srvCmd.Stdout, srvCmd.Stderr = os.Stdout, os.Stderr
    if err := srvCmd.Start(); err != nil { t.Fatalf("srv: %v", err) }
    time.Sleep(300 * time.Millisecond)

    serverURL := fmt.Sprintf("http://127.0.0.1:%d", srvPort)
    clientCmd := exec.CommandContext(ctx, clientBin, "--server", serverURL, "--subdomain", "public", "--port", localPort)
    clientCmd.Stdout, clientCmd.Stderr = os.Stdout, os.Stderr
    if err := clientCmd.Start(); err != nil { t.Fatalf("cli: %v", err) }
    time.Sleep(500 * time.Millisecond)

    req, _ := http.NewRequest("GET", serverURL+"/", nil)
    req.Host = "public.example.com"
    resp, err := http.DefaultClient.Do(req)
    if err != nil { t.Fatalf("request: %v", err) }
    body, _ := io.ReadAll(resp.Body)
    if string(body) != "pong" { t.Fatalf("body: %s", body) }
}
