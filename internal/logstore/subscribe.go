package logstore



// subscription support
func (s *Store) Subscribe() (<-chan Entry, func()) {
    ch := make(chan Entry, 100)
    s.mu.Lock()
    if s.subs == nil {
        s.subs = make([]chan Entry, 0)
    }
    s.subs = append(s.subs, ch)
    s.mu.Unlock()

    cancel := func() {
        s.mu.Lock()
        defer s.mu.Unlock()
        for i, c := range s.subs {
            if c == ch {
                s.subs = append(s.subs[:i], s.subs[i+1:]...)
                close(c)
                break
            }
        }
    }
    return ch, cancel
}

// broadcast helper
func (s *Store) broadcast(e Entry) {
    for _, ch := range s.subs {
        select {
        case ch <- e:
        default:
        }
    }
}
