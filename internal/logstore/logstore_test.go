package logstore

import "testing"

func TestCircularBuffer(t *testing.T) {
    s := New(3)
    s.Add(Entry{ID: "1"})
    s.Add(Entry{ID: "2"})
    s.Add(Entry{ID: "3"})
    if len(s.All()) != 3 {
        t.Fatalf("expected 3 entries")
    }
    s.Add(Entry{ID: "4"})
    all := s.All()
    if len(all) != 3 || all[0].ID != "2" || all[2].ID != "4" {
        t.Fatalf("circular logic wrong: %+v", all)
    }
}
