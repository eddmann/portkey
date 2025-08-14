package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
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


type Client struct {
    conn    *websocket.Conn
    pending sync.Map // id -> chan tunnel.Response
}

var (
    port = flag.Int("port", 8080, "HTTP port to listen on")
    authFile = flag.String("auth-file", "", "Path to auth token YAML file (optional)")
    httpsEnabled = flag.Bool("https", false, "Enable embedded Caddy for TLS")
    domain = flag.String("domain", "localhost", "Base domain of the server")
    caddyEmail = flag.String("caddy-email", "", "Email for Let's Encrypt account")
    enableWebUI = flag.Bool("enable-web-ui", false, "Enable Web UI and live request logging")
    logStoreType = flag.String("log-store", "memory", "Log store backend (memory|sqlite)")
    logDBPath   = flag.String("log-db", "logs.db", "SQLite database file when --log-store=sqlite")
    logRetention = flag.Int("log-retention", 0, "Retention days for SQLite logs (0=keep forever)")
)

func main() {
    flag.Parse()

    if *domain == "" {
        log.Fatal("--domain must be set and non-empty")
    }

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

    normalizeHost := func(host string) string {
        if h, _, err := net.SplitHostPort(host); err == nil {
            return h
        }
        return host
    }

    // On-demand TLS allowlist endpoint: approve cert only for registered subdomains (and apex)
    mux.HandleFunc("/allow-host", func(w http.ResponseWriter, r *http.Request) {
        h := normalizeHost(r.URL.Query().Get("host"))
        d := *domain
        if h == d {
            w.WriteHeader(http.StatusOK)
            return
        }
        // Only allow if exact subdomain is currently registered
        if strings.HasSuffix(h, "."+d) {
            sub := strings.TrimSuffix(h, "."+d)
            if _, ok := reg.Lookup(sub); ok {
                w.WriteHeader(http.StatusOK)
                return
            }
        }
        http.Error(w, "forbidden", http.StatusForbidden)
        return
    })

    isRootHost := func(host string) bool {
        return normalizeHost(host) == *domain
    }

    proxy := func(w http.ResponseWriter, r *http.Request) {
        sub := strings.TrimSuffix(normalizeHost(r.Host), "."+*domain)
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

        respCh := make(chan tunnel.Response, 1)
        client.pending.Store(id, respCh)
        defer client.pending.Delete(id)

        if err := client.conn.WriteJSON(reqMsg); err != nil {
            http.Error(w, "tunnel write error", http.StatusBadGateway)
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
            http.Error(w, "tunnel timeout", http.StatusGatewayTimeout)
        }
    }

    if *enableWebUI {
        uiDir := "../webui"
        if _, err := os.Stat(uiDir); os.IsNotExist(err) {
            uiDir = "/webui"
        }
        fs := http.FileServer(http.Dir(uiDir))
        mux.Handle("/ui/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !isRootHost(r.Host) { proxy(w, r); return }
            http.StripPrefix("/ui/", fs).ServeHTTP(w, r)
        }))
    }

    if *enableWebUI {
        mux.HandleFunc("/api/requests", func(w http.ResponseWriter, r *http.Request) {
            if !isRootHost(r.Host) { proxy(w, r); return }
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
            if !isRootHost(r.Host) { proxy(w, r); return }
            if mgr != nil && mgr.Role(r.URL.Query().Get("token")) != "admin" {
                http.Error(w, "forbidden", http.StatusForbidden)
                return
            }
            w.Header().Set("Content-Type", "application/json")
            subs := reg.Subdomains()
            json.NewEncoder(w).Encode(subs)
        })
    }

    if *enableWebUI {
        mux.HandleFunc("/api/ws", func(w http.ResponseWriter, r *http.Request) {
            if !isRootHost(r.Host) { proxy(w, r); return }
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

    mux.HandleFunc("/", proxy)

    listenAddr := ":" + strconv.Itoa(*port)
    if *httpsEnabled {
        listenAddr = "127.0.0.1:8081"
        ctx := context.Background()
        ask := "http://" + listenAddr + "/allow-host"
        if err := caddysetup.Start(ctx, ":"+strconv.Itoa(*port), listenAddr, *domain, *caddyEmail, ask); err != nil {
            log.Fatalf("caddy start: %v", err)
        }
    }

    log.Printf("portkey-server listening on %s", listenAddr)
    if err := http.ListenAndServe(listenAddr, mux); err != nil {
        log.Fatal(err)
    }
}
