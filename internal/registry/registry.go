package registry

import "sync"

type ClientConn interface{}

type Registry struct {
    
    mu sync.RWMutex
    m  map[string]ClientConn
}

func New() *Registry {
    return &Registry{
        m: make(map[string]ClientConn),
    }
}

func (r *Registry) Register(sub string, c ClientConn) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.m[sub] = c
}

func (r *Registry) Lookup(sub string) (ClientConn, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    c, ok := r.m[sub]
    return c, ok
}

func (r *Registry) Remove(sub string) {
    r.mu.Lock()
    defer r.mu.Unlock()
    delete(r.m, sub)
}

// Subdomains returns currently registered sub-domain names.
func (r *Registry) Subdomains() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    keys := make([]string, 0, len(r.m))
    for k := range r.m {
        keys = append(keys, k)
    }
    return keys
}
