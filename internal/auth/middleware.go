package auth

import (
	"net/http"
)

// RequireRole returns a middleware that ensures the request's token has the given role.
// Token is looked up from "X-Auth-Token" header or "token" query parameter.
func RequireRole(role string, mgr *Manager) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("X-Auth-Token")
            if token == "" {
                token = r.URL.Query().Get("token")
            }
            if mgr.Role(token) != role {
                http.Error(w, "forbidden", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
