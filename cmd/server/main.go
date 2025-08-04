package main

import (
	"context"
	"encoding/json"
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
	"portkey/internal/logstore"
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
    enableWebUI = flag.Bool("enable-web-ui", false, "Enable Web UI and live request logging")
    logStoreType = flag.String("log-store", "memory", "Log store backend (memory|sqlite)")
    logDBPath   = flag.String("log-db", "logs.db", "SQLite database file when --log-store=sqlite")
    logRetention = flag.Int("log-retention", 0, "Retention days for SQLite logs (0=keep forever)")
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
    memStore := logstore.New(1000)
    var sqlStore *logstore.SQLite
    if *logStoreType == "sqlite" {
        s, err := logstore.NewSQLite(*logDBPath)
        if err != nil { log.Fatalf("sqlite: %v", err) }
        sqlStore = s
        log.Printf("SQLite logstore enabled (%s)", *logDBPath)
        if *logRetention > 0 {
            go func() {
                ticker := time.NewTicker(12 * time.Hour)
                for {
                    cutoff := time.Now().AddDate(0,0,-*logRetention)
                    sqlStore.PurgeOlderThan(cutoff)
                    <-ticker.C
                }
            }()
            log.Printf("log retention: %d days", *logRetention)
        }
    }
    storeEnabled := *enableWebUI || sqlStore != nil

    mux := http.NewServeMux()
    // Web UI static files
    if *enableWebUI {
        fs := http.FileServer(http.Dir("webui"))
        mux.Handle("/ui/", http.StripPrefix("/ui/", fs))
    }
    // REST admin APIs
    if *enableWebUI {
        mux.HandleFunc("/api/requests", func(w http.ResponseWriter, r *http.Request) {
            if mgr != nil && mgr.Role(r.URL.Query().Get("token")) != "admin" {
                http.Error(w, "forbidden", http.StatusForbidden)
                return
            }
            if !storeEnabled {
                http.Error(w, "logstore disabled", http.StatusServiceUnavailable)
                return
            }
            id := strings.TrimPrefix(r.URL.Path, "/api/requests/")
            w.Header().Set("Content-Type", "application/json")
            if id != "" && id != "/api/requests" {
                if e, ok := memStore.Get(id); ok {
                    json.NewEncoder(w).Encode(e)
                    return
                }
                if sqlStore != nil {
                    entries, _ := sqlStore.All()
                    for _, ent := range entries { if ent.ID == id { json.NewEncoder(w).Encode(ent); return } }
                }
                http.NotFound(w, r)
                return
            }
            var entries []logstore.Entry
            if sqlStore != nil {
                entries, _ = sqlStore.All()
            } else {
                entries = memStore.All()
            }
            json.NewEncoder(w).Encode(entries)
        })
        mux.HandleFunc("/api/tunnels", func(w http.ResponseWriter, r *http.Request) {
            if mgr != nil && mgr.Role(r.URL.Query().Get("token")) != "admin" {
                http.Error(w, "forbidden", http.StatusForbidden)
                return
            }
            w.Header().Set("Content-Type", "application/json")
            subs := reg.Subdomains()
            json.NewEncoder(w).Encode(subs)
        })
    }

    // Web UI websocket endpoint
    if *enableWebUI {
        mux.HandleFunc("/api/ws", func(w http.ResponseWriter, r *http.Request) {
            if mgr != nil && mgr.Role(r.URL.Query().Get("token")) != "admin" {
                http.Error(w, "forbidden", http.StatusForbidden)
                return
            }
            up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
            ws, err := up.Upgrade(w, r, nil)
            if err != nil { return }
            ch, cancel := memStore.Subscribe()
            defer cancel()
            for entry := range ch {
                ws.WriteJSON(entry)
            }
        })
    }

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
            entry := logstore.Entry{ID: id, Subdomain: sub, Method: r.Method, Path: r.URL.RequestURI(), Status: resp.Status, Timestamp: time.Now(), Headers: func() map[string]string {
                    h := make(map[string]string)
                    for k, v := range r.Header {
                        h[k] = strings.Join(v, ";")
                    }
                    return h
                }(),
                Body: string(reqMsg.Body)}
            memStore.Add(entry)
            if sqlStore != nil {
                sqlStore.Add(entry)
            }
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
