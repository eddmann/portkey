package auth

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
    "path"
)

type TokenEntry struct {
    Token      string   `yaml:"token"`
    Subdomains []string `yaml:"subdomains"`
    Role       string   `yaml:"role"`
}

type Manager struct {
    entries map[string]TokenEntry // token -> entry
}

// NewManagerFromFile loads a YAML file into a token manager.
func NewManagerFromFile(path string) (*Manager, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read auth file: %w", err)
    }
    var cfg struct {
        Tokens []TokenEntry `yaml:"tokens"`
    }
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("yaml: %w", err)
    }
    m := &Manager{entries: make(map[string]TokenEntry)}
    for _, t := range cfg.Tokens {
        m.entries[t.Token] = t
    }
    return m, nil
}

// Validate checks that the token exists and its allowed patterns match the subdomain.
// Supports:
//   * exact match ("project1")
//   * global wildcard "*"
//   * shell-style patterns with '*' such as "project1-*-foo".
func (m *Manager) Validate(token, sub string) bool {
    if token == "" {
        return false
    }
    e, ok := m.entries[token]
    if !ok {
        return false
    }
    for _, pattern := range e.Subdomains {
        if pattern == "*" {
            return true
        }
        if pattern == sub {
            return true
        }
        if ok, _ := path.Match(pattern, sub); ok {
            return true
        }
    }
    return false
}

// Role returns role for token or empty string if not found.
func (m *Manager) Role(token string) string {
    if e, ok := m.entries[token]; ok {
        return e.Role
    }
    return ""
}

var ErrUnauthorized = errors.New("unauthorized")
