package middleware

import (
	"net/http"

	"github.com/rs/zerolog/hlog"
)

// MaxBytesReader limits the size of request bodies
func MaxBytesReader(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l := hlog.FromRequest(r)
			
			// Limit request body size
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			
			next.ServeHTTP(w, r)
			
			// Check if body was too large
			if r.ContentLength > maxBytes {
				l.Warn().Int64("content_length", r.ContentLength).Int64("max_bytes", maxBytes).Msg("Request body too large")
			}
		})
	}
}
