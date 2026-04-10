package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rs/zerolog/hlog"
)

func writeJSONUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Unauthorized"}) //nolint:errcheck
}

func extractBearer(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", false
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", false
	}
	return parts[1], true
}

func BearerTokenAuth(expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l := hlog.FromRequest(r)

			if expectedToken == "" {
				l.Error().Msg("Bearer token is not configured")
				writeJSONUnauthorized(w)
				return
			}

			key, ok := extractBearer(r)
			if !ok {
				l.Warn().Msg("Missing or malformed Authorization header")
				writeJSONUnauthorized(w)
				return
			}

			if subtle.ConstantTimeCompare([]byte(key), []byte(expectedToken)) != 1 {
				l.Warn().Msg("Invalid bearer token")
				writeJSONUnauthorized(w)
				return
			}

			l.Debug().Msg("Bearer token validated successfully")
			next.ServeHTTP(w, r)
		})
	}
}
