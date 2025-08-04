package auth

import "testing"

func TestValidate(t *testing.T) {
    m := &Manager{entries: map[string]TokenEntry{
        "abc": {Token: "abc", Subdomains: []string{"project1"}, Role: "user"},
        "admin": {Token: "admin", Subdomains: []string{"*"}, Role: "admin"},
    }}

    if !m.Validate("abc", "project1") {
        t.Fatalf("expected valid")
    }
    if m.Validate("abc", "other") {
        t.Fatalf("expected invalid subdomain")
    }
    if !m.Validate("admin", "whatever") {
        t.Fatalf("admin should allow any subdomain")
    }
}
