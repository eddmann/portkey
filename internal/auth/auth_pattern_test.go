package auth

import "testing"

func TestPatternValidation(t *testing.T) {
    m := &Manager{entries: map[string]TokenEntry{
        "t1": {Token: "t1", Subdomains: []string{"project1-*"}},
    }}

    if !m.Validate("t1", "project1-abc") {
        t.Errorf("pattern failed to match")
    }
    if m.Validate("t1", "project2-abc") {
        t.Errorf("unexpected match")
    }
}
