package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/hlog"
)

// HealthChecker provides health check functionality.
type HealthChecker struct {
	redis    *redis.Client
	rabbitmq *amqp.Connection
	deps     map[string]func(context.Context) error
}

// HealthStatus represents the health status of the application.
type HealthStatus struct {
	Status       string            `json:"status"`
	Timestamp    time.Time         `json:"timestamp"`
	Dependencies map[string]string `json:"dependencies"`
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(rdb *redis.Client, rabbitmq *amqp.Connection) *HealthChecker {
	hc := &HealthChecker{
		redis:    rdb,
		rabbitmq: rabbitmq,
		deps:     make(map[string]func(context.Context) error),
	}

	if rdb != nil {
		hc.RegisterDependency("redis", hc.checkRedis)
	}
	if rabbitmq != nil {
		hc.RegisterDependency("rabbitmq", hc.checkRabbitMQ)
	}

	return hc
}

// RegisterDependency registers a custom dependency check.
func (hc *HealthChecker) RegisterDependency(name string, checkFunc func(context.Context) error) {
	hc.deps[name] = checkFunc
}

func (hc *HealthChecker) checkRedis(ctx context.Context) error {
	if hc.redis == nil {
		return nil
	}
	return hc.redis.Ping(ctx).Err()
}

func (hc *HealthChecker) checkRabbitMQ(ctx context.Context) error {
	if hc.rabbitmq == nil {
		return nil
	}
	if hc.rabbitmq.IsClosed() {
		return errors.New("rabbitmq connection closed")
	}
	return nil
}

// Check performs health check on all dependencies.
func (hc *HealthChecker) Check(ctx context.Context) HealthStatus {
	status := HealthStatus{
		Status:       "healthy",
		Timestamp:    time.Now(),
		Dependencies: make(map[string]string),
	}

	for name, checkFunc := range hc.deps {
		checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := checkFunc(checkCtx)
		cancel()

		if err != nil {
			status.Dependencies[name] = "unhealthy: " + err.Error()
			status.Status = "unhealthy"
		} else {
			status.Dependencies[name] = "healthy"
		}
	}

	return status
}

// Handler returns an HTTP handler for the full dependency health check.
func (hc *HealthChecker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := hlog.FromRequest(r)
		ctx := r.Context()

		status := hc.Check(ctx)

		w.Header().Set("Content-Type", "application/json")
		if status.Status == "healthy" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		if err := json.NewEncoder(w).Encode(status); err != nil {
			l.Error().Err(err).Msg("Failed to encode health status")
		}
	}
}

// ReadinessHandler returns a readiness probe handler.
func (hc *HealthChecker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		status := hc.Check(ctx)

		w.Header().Set("Content-Type", "application/json")
		if status.Status == "healthy" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(map[string]string{"status": status.Status})
	}
}

// LivenessHandler returns a liveness probe handler.
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	}
}
