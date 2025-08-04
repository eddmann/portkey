package logstore

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "modernc.org/sqlite"
)

type SQLite struct {
    db *sql.DB
}

func NewSQLite(path string) (*SQLite, error) {
    db, err := sql.Open("sqlite", path)
    if err != nil { return nil, err }
    if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS logs (
        id TEXT PRIMARY KEY,
        subdomain TEXT,
        method TEXT,
        path TEXT,
        status INTEGER,
        headers TEXT,
        body TEXT,
        ts INTEGER
    )`); err != nil { return nil, err }
    return &SQLite{db: db}, nil
}

func (s *SQLite) Add(e Entry) error {
    _, err := s.db.Exec(`INSERT INTO logs (id, subdomain, method, path, status, headers, body, ts) VALUES (?,?,?,?,?,?,?,?)`,
        e.ID, e.Subdomain, e.Method, e.Path, e.Status, marshalJSON(e.Headers), e.Body, e.Timestamp.Unix())
    return err
}

func (s *SQLite) All() ([]Entry, error) {
    rows, err := s.db.Query(`SELECT id, subdomain, method, path, status, headers, body, ts FROM logs ORDER BY ts DESC`)
    if err != nil { return nil, err }
    defer rows.Close()
    var out []Entry
    for rows.Next() {
        var e Entry
        var headers string
        var ts int64
        if err := rows.Scan(&e.ID, &e.Subdomain, &e.Method, &e.Path, &e.Status, &headers, &e.Body, &ts); err != nil { return nil, err }
        e.Headers = unmarshalJSON(headers)
        e.Timestamp = time.Unix(ts,0)
        out = append(out, e)
    }
    return out, nil
}

func marshalJSON(h map[string]string) string {
    b, _ := json.Marshal(h)
    return string(b)
}
func unmarshalJSON(s string) map[string]string { var h map[string]string; _=json.Unmarshal([]byte(s),&h); return h }
