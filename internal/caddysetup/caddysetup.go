package caddysetup

import (
	"context"
	"encoding/json"
	"log"

	caddy "github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
)

// Start launches an embedded Caddy instance that proxies from listenAddr
// to upstream (e.g., 127.0.0.1:8081). Domain/email are used for automatic TLS.
func Start(ctx context.Context, listenAddr, upstream, domain, email string) error {
    cfg := map[string]any{
        "apps": map[string]any{
            "http": map[string]any{
                "servers": map[string]any{
                    "srv0": map[string]any{
                        "listen": []string{listenAddr},
                        "routes": []any{
                            map[string]any{
                                "handle": []any{
                                    map[string]any{
                                        "handler": "reverse_proxy",
                                        "upstreams": []any{
                                            map[string]any{"dial": upstream},
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
                "automatic_https": map[string]any{
                    "disable": false,
                },
            },
        },
        "certificates": map[string]any{
            "automate": []string{domain},
        },
        "storage": map[string]any{
            "module": "memory",
        },
        "email": email,
    }

    raw, _ := json.Marshal(cfg)

    caddyCfg, err := caddyconfig.LoadJSON(raw, "json")
    if err != nil {
        return err
    }

    instance, err := caddy.Start(caddyCfg)
    if err != nil {
        return err
    }

    go func() {
        <-ctx.Done()
        instance.Stop()
    }()

    log.Printf("Caddy started on %s, proxy -> %s", listenAddr, upstream)
    return nil
}
