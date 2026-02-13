package middleware

import (
	"net/http"
)

// MaxBytesReader limits the size of request bodies
func MaxBytesReader(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Limit request body size
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			
			next.ServeHTTP(w, r)
		})
	}
}
