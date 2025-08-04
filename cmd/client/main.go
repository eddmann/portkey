package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"

	"portkey/internal/tunnel"
)

var (
    port     = flag.Int("port", 3000, "Local port to expose")
    host     = flag.String("host", "localhost", "Local host running service")
    server   = flag.String("server", "http://localhost:8080", "Portkey server URL")
    subdomain = flag.String("subdomain", "myapp", "Requested subdomain")
    authToken = flag.String("auth-token", "", "Auth token for server")
)

func main() {
    flag.Parse()

    u, err := url.Parse(*server)
    if err != nil {
        log.Fatalf("invalid server url: %v", err)
    }

    log.Printf("Connecting to %s for subdomain %s (forwarding %s:%d)", *server, *subdomain, *host, *port)

    // Dial websocket
    wsURL := u
    wsURL.Scheme = strings.ReplaceAll(wsURL.Scheme, "http", "ws")
    wsURL.Path = "/connect"
    q := wsURL.Query()
    q.Set("subdomain", *subdomain)
    if *authToken != "" {
        q.Set("token", *authToken)
    }
    wsURL.RawQuery = q.Encode()

    conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
    if err != nil {
        log.Fatalf("dial error: %v", err)
    }
    defer conn.Close()

    log.Printf("Tunnel established, waiting for requests ...")

    for {
        var req tunnel.Request
        if err := conn.ReadJSON(&req); err != nil {
            log.Fatalf("read: %v", err)
        }

        go handleRequest(conn, req)
    }
}

func handleRequest(conn *websocket.Conn, req tunnel.Request) {
    // Forward to local server
    target := fmt.Sprintf("http://%s:%d%s", *host, *port, req.Path)
    httpReq, err := http.NewRequest(req.Method, target, bytes.NewReader(req.Body))
    if err != nil {
        log.Printf("build req: %v", err)
        return
    }
    for k, v := range req.Headers {
        httpReq.Header.Set(k, v)
    }

    resp, err := http.DefaultClient.Do(httpReq)
    if err != nil {
        log.Printf("local request error: %v", err)
        sendError(conn, req.ID, err)
        return
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    headers := make(map[string]string)
    for k, v := range resp.Header {
        headers[k] = strings.Join(v, ";")
    }
    resMsg := tunnel.Response{
        ID:     req.ID,
        Status: resp.StatusCode,
        Headers: headers,
        Body:   body,
    }
    if err := conn.WriteJSON(resMsg); err != nil {
        log.Printf("write back: %v", err)
    }
}

func sendError(conn *websocket.Conn, id string, err error) {
    res := tunnel.Response{
        ID:     id,
        Status: 502,
        Headers: map[string]string{"Content-Type": "text/plain"},
        Body:   []byte(err.Error()),
    }
    conn.WriteJSON(res)
}
