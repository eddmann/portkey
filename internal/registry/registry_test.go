package registry

import (
	"sync"
	"testing"
)

type dummyConn struct{}

func TestRegisterLookupRemove(t *testing.T) {
    reg := New()

    conn := &dummyConn{}
    reg.Register("foo", conn)

    if got, ok := reg.Lookup("foo"); !ok || got != conn {
        t.Fatalf("expected to find conn, got %v ok=%v", got, ok)
    }

    reg.Remove("foo")
    if _, ok := reg.Lookup("foo"); ok {
        t.Fatalf("expected conn to be removed")
    }
}

func TestConcurrencySafety(t *testing.T) {
    reg := New()
    const n = 1000

    conn := &dummyConn{}
    wg := sync.WaitGroup{}
    wg.Add(n * 2)

    // concurrent writers
    for i := 0; i < n; i++ {
        go func(i int) {
            defer wg.Done()
            reg.Register(string(rune(i)), conn)
        }(i)
    }
    // concurrent readers
    for i := 0; i < n; i++ {
        go func(i int) {
            defer wg.Done()
            reg.Lookup(string(rune(i)))
        }(i)
    }

    wg.Wait()
}
