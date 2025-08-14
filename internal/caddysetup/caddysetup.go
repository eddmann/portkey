package caddysetup

import (
	"context"
	"encoding/json"
	"log"

	caddy "github.com/caddyserver/caddy/v2"
	// Register standard modules (http, tls, reverse_proxy, file storage, etc.)
	_ "github.com/caddyserver/caddy/v2/modules/standard"
)

// Start launches an embedded Caddy instance that proxies from listenAddr
// to upstream (e.g., 127.0.0.1:8081). Domain/email are used for automatic TLS.
func Start(ctx context.Context, listenAddr, upstream, domain, email, askURL string) error {
    // Build TLS automation policy, using internal issuer for localhost/dev
    policy := map[string]any{
        "on_demand": true,
    }
    if domain == "localhost" {
        policy["issuers"] = []any{
            map[string]any{"module": "internal"},
        }
    }

    cfg := map[string]any{
        "apps": map[string]any{
            "tls": map[string]any{
                "automation": map[string]any{
                    "policies": []any{policy},
                    "on_demand": map[string]any{
                        "ask": askURL,
                    },
                },
            },
            "http": map[string]any{
                "servers": map[string]any{
                    "srv0": map[string]any{
                        "listen": []string{listenAddr},
                        "routes": []any{
                            map[string]any{
                                "match": []any{map[string]any{"host": []string{domain, "*." + domain}}},
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
            },
        },
        "email": email,
    }

    raw, _ := json.Marshal(cfg)

    var conf caddy.Config
    if err := json.Unmarshal(raw, &conf); err != nil {
        return err
    }

    go func() {
        <-ctx.Done()
        caddy.Stop()
    }()

    if err := caddy.Run(&conf); err != nil {
        return err
    }

    log.Printf("Caddy started on %s, proxy -> %s", listenAddr, upstream)
    return nil
}
