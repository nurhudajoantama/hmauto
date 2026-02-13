package middleware

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog/hlog"
)

// APIKeyAuth middleware validates API key from Authorization header
func APIKeyAuth(validAPIKeys map[string]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l := hlog.FromRequest(r)

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				l.Warn().Msg("Missing Authorization header")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Expected format: "Bearer <api-key>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				l.Warn().Msg("Invalid Authorization header format")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			apiKey := parts[1]
			if !validAPIKeys[apiKey] {
				l.Warn().Msg("Invalid API key")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			l.Debug().Msg("API key validated successfully")
			next.ServeHTTP(w, r)
		})
	}
}
