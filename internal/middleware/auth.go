package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/nurhudajoantama/hmauto/internal/apikey"
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

// APIKeyAuth middleware validates API keys via the apikey.Store (Redis-backed).
func APIKeyAuth(store apikey.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l := hlog.FromRequest(r)

			key, ok := extractBearer(r)
			if !ok {
				l.Warn().Msg("Missing or malformed Authorization header")
				writeJSONUnauthorized(w)
				return
			}

			valid, err := store.ValidateKey(r.Context(), key)
			if err != nil {
				l.Error().Err(err).Msg("API key validation error")
				writeJSONUnauthorized(w)
				return
			}
			if !valid {
				l.Warn().Msg("Invalid API key")
				writeJSONUnauthorized(w)
				return
			}

			l.Debug().Msg("API key validated successfully")
			next.ServeHTTP(w, r)
		})
	}
}

// AdminKeyAuth middleware validates the admin key against the config value only (no Redis).
func AdminKeyAuth(adminKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l := hlog.FromRequest(r)

			key, ok := extractBearer(r)
			if !ok {
				l.Warn().Msg("Missing or malformed Authorization header (admin)")
				writeJSONUnauthorized(w)
				return
			}

			if key != adminKey {
				l.Warn().Msg("Invalid admin key")
				writeJSONUnauthorized(w)
				return
			}

			l.Debug().Msg("Admin key validated successfully")
			next.ServeHTTP(w, r)
		})
	}
}
