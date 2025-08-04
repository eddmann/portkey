package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

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
)

func main() {
    flag.Parse()

    reg := registry.New()

    mux := http.NewServeMux()

    // WebSocket endpoint for clients to register their tunnel
    mux.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
        sub := r.URL.Query().Get("subdomain")
        if sub == "" {
            http.Error(w, "missing subdomain", http.StatusBadRequest)
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

    log.Printf("portkey-server listening on %s", *addr)
    if err := http.ListenAndServe(*addr, mux); err != nil {
        log.Fatal(err)
    }
}
