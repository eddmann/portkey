package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"portkey/internal/auth"
	"portkey/internal/caddysetup"
	"portkey/internal/registry"
	"portkey/internal/tunnel"
)

// Client wraps a websocket connection and outstanding request map
type Client struct {
    conn    *websocket.Conn
    pending sync.Map // id -> chan tunnel.Response
}


var (
    addr = flag.String("addr", ":8080", "HTTP listen address")
    authFile = flag.String("auth-file", "", "Path to auth token YAML file (optional)")
    useCaddy = flag.Bool("use-caddy", false, "Enable embedded Caddy for TLS")
    caddyDomain = flag.String("caddy-domain", "", "Domain to get TLS cert for")
    caddyEmail = flag.String("caddy-email", "", "Email for Let's Encrypt account")
)

func main() {
    flag.Parse()

    var mgr *auth.Manager
    if *authFile != "" {
        var err error
        mgr, err = auth.NewManagerFromFile(*authFile)
        if err != nil {
            log.Fatalf("auth load: %v", err)
        }
        log.Printf("auth enabled (%s)", *authFile)
    } else {
        log.Printf("auth disabled (no auth-file provided)")
    }

    reg := registry.New()

    mux := http.NewServeMux()

    // WebSocket endpoint for clients to register their tunnel
    mux.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
        sub := r.URL.Query().Get("subdomain")
        token := r.URL.Query().Get("token")
        if sub == "" {
            http.Error(w, "missing subdomain", http.StatusBadRequest)
            return
        }
        if mgr != nil && !mgr.Validate(token, sub) {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
        ws, err := up.Upgrade(w, r, nil)
        if err != nil {
            log.Printf("upgrade error: %v", err)
            return
        }

        client := &Client{conn: ws}
        reg.Register(sub, client)
        log.Printf("subdomain %s registered", sub)

        // read loop
        go func() {
            defer func() {
                reg.Remove(sub)
                ws.Close()
                log.Printf("subdomain %s disconnected", sub)
            }()
            for {
                var resp tunnel.Response
                if err := ws.ReadJSON(&resp); err != nil {
                    log.Printf("read error: %v", err)
                    return
                }
                if chVal, ok := client.pending.Load(resp.ID); ok {
                    ch := chVal.(chan tunnel.Response)
                    ch <- resp
                }
            }
        }()
    })

    // Wild-card HTTP handler – proxies requests to matching tunnel
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        host := r.Host
        sub := strings.Split(host, ".")[0]
        cVal, ok := reg.Lookup(sub)
        if !ok {
            http.NotFound(w, r)
            return
        }
        client := cVal.(*Client)

        id := uuid.New().String()
        reqMsg := tunnel.Request{
            ID:     id,
            Method: r.Method,
            Path:   r.URL.RequestURI(),
            Headers: func() map[string]string {
                h := make(map[string]string)
                for k, v := range r.Header {
                    h[k] = strings.Join(v, ";")
                }
                return h
            }(),
        }
        if b, err := io.ReadAll(r.Body); err == nil {
            reqMsg.Body = b
        }

        // create response channel
        respCh := make(chan tunnel.Response, 1)
        client.pending.Store(id, respCh)
        defer client.pending.Delete(id)

        if err := client.conn.WriteJSON(reqMsg); err != nil {
            http.Error(w, "tunnel write error", 502)
            return
        }

        select {
        case resp := <-respCh:
            for k, v := range resp.Headers {
                w.Header().Set(k, v)
            }
            w.WriteHeader(resp.Status)
            w.Write(resp.Body)
        case <-time.After(30 * time.Second):
            http.Error(w, "tunnel timeout", 504)
        }
    })

    listenAddr := *addr
    if *useCaddy {
        // Shift internal mux to 127.0.0.1:8081 and let Caddy listen on *addr
        listenAddr = ":8081"
        ctx := context.Background()
        domain := *caddyDomain
        if domain == "" {
            log.Fatal("--caddy-domain required when --use-caddy is set")
        }
        if err := caddysetup.Start(ctx, *addr, "127.0.0.1"+listenAddr, domain, *caddyEmail); err != nil {
            log.Fatalf("caddy start: %v", err)
        }
    }

    log.Printf("portkey-server listening on %s", listenAddr)
    if err := http.ListenAndServe(listenAddr, mux); err != nil {
        log.Fatal(err)
    }
}
