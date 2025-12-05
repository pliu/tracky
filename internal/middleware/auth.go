package middleware

import (
	"context"
	"net/http"
	"strings"

	"tracky/internal/auth"
)

// Auth validates the signed cookie and adds user ID to context
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for public endpoints
		if isPublicEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("auth_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userID, err := auth.ValidateSignedCookie(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add user ID to context
		ctx := context.WithValue(r.Context(), auth.UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublicEndpoint(path string) bool {
	// Exact match paths
	exactPaths := []string{"/", "/api/signup", "/api/login"}
	for _, p := range exactPaths {
		if path == p {
			return true
		}
	}
	// Prefix match paths
	prefixPaths := []string{"/static/"}
	for _, p := range prefixPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
