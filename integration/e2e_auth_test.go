package integration

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// findFreePort asks the kernel for a free open port that is ready to use.
func findFreePort() (int, error) {
    l, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        return 0, err
    }
    defer l.Close()
    return l.Addr().(*net.TCPAddr).Port, nil
}

func buildBinary(t *testing.T, pkg, out string) {
    cmd := exec.Command("go", "build", "-o", out, pkg)
    cmd.Env = append(os.Environ(), "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
    if outBytes, err := cmd.CombinedOutput(); err != nil {
        t.Fatalf("build %s: %v\n%s", pkg, err, string(outBytes))
    }
}

func TestBlackboxTunnel(t *testing.T) {
    tempDir := t.TempDir()

    serverBin := filepath.Join(tempDir, "portkey-server")
    clientBin := filepath.Join(tempDir, "portkey-client")

    // Build binaries
    buildBinary(t, "../cmd/server", serverBin)
    buildBinary(t, "../cmd/client", clientBin)

    // Dummy local app returning "pong"
    localApp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("pong"))
    }))
    defer localApp.Close()
    localPort := strings.Split(localApp.URL, ":")[2]

    // Pick random port for server
    serverPort, err := findFreePort()
    if err != nil {
        t.Fatalf("free port: %v", err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Start server process
    authPath := filepath.Join("..", "auth.yaml")
    serverCmd := exec.CommandContext(ctx, serverBin, "-addr", fmt.Sprintf(":%d", serverPort), "-auth-file", authPath)
    serverCmd.Stdout = os.Stdout
    serverCmd.Stderr = os.Stderr
    if err := serverCmd.Start(); err != nil {
        t.Fatalf("start server: %v", err)
    }

    // Wait a moment for server to listen
    time.Sleep(300 * time.Millisecond)

    // Start CLI process
    serverURL := fmt.Sprintf("http://127.0.0.1:%d", serverPort)
    clientCmd := exec.CommandContext(ctx, clientBin, "--server", serverURL, "--subdomain", "myapp", "--port", localPort, "--auth-token", "admin456")
    clientCmd.Stdout = os.Stdout
    clientCmd.Stderr = os.Stderr
    if err := clientCmd.Start(); err != nil {
        t.Fatalf("start cli: %v", err)
    }

    // give time for registration
    time.Sleep(500 * time.Millisecond)

    // Perform request through tunnel
    req, _ := http.NewRequest("GET", serverURL+"/", nil)
    req.Host = "myapp.example.com"
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        t.Fatalf("request err: %v", err)
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    if string(body) != "pong" {
        t.Fatalf("unexpected response body: %s", string(body))
    }
}
