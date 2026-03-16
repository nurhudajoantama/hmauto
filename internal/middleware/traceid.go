package middleware

import (
	"net/http"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// TraceIDMiddleware injects the OTEL trace ID into the zerolog context.
func TraceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			traceID := span.SpanContext().TraceID().String()
			logger := zerolog.Ctx(ctx).With().Str("trace_id", traceID).Logger()
			ctx = logger.WithContext(ctx)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
