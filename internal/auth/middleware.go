package auth

import (
	"net/http"
	"strings"
)

// Middleware returns an http.Handler that gates access based on auth state.
func Middleware(am *Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Auth disabled entirely
		if am.NoAuth() {
			next.ServeHTTP(w, r)
			return
		}

		path := r.URL.Path

		// Always allow static assets
		if strings.HasPrefix(path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		// No password set yet — only allow setup-related routes
		if !am.HasPassword() {
			if path == "/setup" || path == "/api/auth/setup" ||
				strings.HasPrefix(path, "/api/keypairs") {
				next.ServeHTTP(w, r)
				return
			}
			// Redirect everything else to setup
			if isAPIRequest(r) {
				http.Error(w, `{"error":"setup required"}`, http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
			return
		}

		// Public auth routes (login page + login API)
		if path == "/login" || path == "/api/auth/login" {
			next.ServeHTTP(w, r)
			return
		}

		// Check session cookie
		if cookie, err := r.Cookie("session"); err == nil {
			if am.ValidSession(cookie.Value) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Check Authorization: Bearer <token>
		if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if am.ValidAPIToken(token) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Not authenticated
		if isAPIRequest(r) || strings.HasPrefix(path, "/ws/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
	})
}

func isAPIRequest(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/api/")
}
