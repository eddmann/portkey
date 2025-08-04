package logstore

import (
	"sync"
	"time"
)

type Entry struct {
    ID        string    `json:"id"`
    Subdomain string    `json:"subdomain"`
    Method    string    `json:"method"`
    Path      string    `json:"path"`
    Status    int       `json:"status"`
    Headers   map[string]string `json:"headers,omitempty"`
    Body      string            `json:"body,omitempty"`
    Timestamp time.Time         `json:"timestamp"`
}

// Store is a fixed-size circular buffer of entries safe for concurrent use.
type Store struct {
    mu   sync.RWMutex
    buf  []Entry
    subs []chan Entry
    size int
    head int
    full bool
}

func New(size int) *Store {
    return &Store{buf: make([]Entry, size), size: size}
}

func (s *Store) Add(e Entry) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.buf[s.head] = e
    s.head = (s.head + 1) % s.size
    if s.head == 0 {
        s.full = true
    }
    s.broadcast(e)
}

func (s *Store) All() []Entry {
    s.mu.RLock()
    defer s.mu.RUnlock()
    var out []Entry
    if s.full {
        out = append(out, s.buf[s.head:]...)
    }
    out = append(out, s.buf[:s.head]...)
    // copy to avoid race
    res := make([]Entry, len(out))
    copy(res, out)
    return res
}

func (s *Store) Get(id string) (Entry, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    for _, e := range s.buf {
        if e.ID == id {
            return e, true
        }
    }
    return Entry{}, false
}
