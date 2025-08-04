package integration

import (
	"context"
	"encoding/json"
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

func TestAPIWithLogs(t *testing.T) {
    tmp := t.TempDir()
    srvBin := filepath.Join(tmp, "srv")
    clientBin := filepath.Join(tmp, "cli")

    buildBinary(t, "../cmd/server", srvBin)
    buildBinary(t, "../cmd/client", clientBin)

    dummy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("pong")) }))
    defer dummy.Close()
    port := strings.Split(dummy.URL, ":")[2]

    portFree, _ := findFreePort()
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    authPath := filepath.Join(".", "auth.yaml")
    srvCmd := exec.CommandContext(ctx, srvBin, "-addr", fmt.Sprintf(":%d", portFree), "-auth-file", authPath, "--enable-web-ui")
    srvCmd.Stdout, srvCmd.Stderr = os.Stdout, os.Stderr
    if err := srvCmd.Start(); err != nil { t.Fatalf("srv: %v", err) }
    time.Sleep(400 * time.Millisecond)

    serverURL := fmt.Sprintf("http://127.0.0.1:%d", portFree)
    clientCmd := exec.CommandContext(ctx, clientBin, "--server", serverURL, "--subdomain", "mylogs", "--port", port, "--auth-token", "admin456")
    clientCmd.Stdout, clientCmd.Stderr = os.Stdout, os.Stderr
    if err := clientCmd.Start(); err != nil { t.Fatalf("cli: %v", err) }
    time.Sleep(600 * time.Millisecond)

    // issue request
    req, _ := http.NewRequest("GET", serverURL+"/", nil)
    req.Host = "mylogs.example.com"
    if _, err := http.DefaultClient.Do(req); err != nil { t.Fatalf("proxy req: %v", err) }

    // fetch logs
    resp, err := http.Get(serverURL + "/api/requests?token=admin456")
    if err != nil { t.Fatalf("api: %v", err) }
    bytes, _ := io.ReadAll(resp.Body)
    var arr []map[string]any
    if err := json.Unmarshal(bytes, &arr); err != nil { t.Fatalf("decode %v body %s", err, string(bytes)) }
    if len(arr) == 0 {
        t.Fatalf("expected logs, got 0")
    }

    // fetch tunnels
    resp2, err := http.Get(serverURL + "/api/tunnels?token=admin456")
    if err != nil { t.Fatalf("tunnels api: %v", err) }
    var tunnels []string
    if err := json.NewDecoder(resp2.Body).Decode(&tunnels); err != nil { t.Fatalf("decode tunnels: %v", err) }
    if len(tunnels) == 0 {
        t.Fatalf("expected tunnels")
    }
}
