package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/hlog"
	"gorm.io/gorm"
)

// HealthChecker provides health check functionality
type HealthChecker struct {
	db           *gorm.DB
	rabbitmq     *amqp.Connection
	dependencies map[string]func(context.Context) error
}

// HealthStatus represents the health status of the application
type HealthStatus struct {
	Status       string            `json:"status"`
	Timestamp    time.Time         `json:"timestamp"`
	Dependencies map[string]string `json:"dependencies"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *gorm.DB, rabbitmq *amqp.Connection) *HealthChecker {
	hc := &HealthChecker{
		db:           db,
		rabbitmq:     rabbitmq,
		dependencies: make(map[string]func(context.Context) error),
	}

	// Register built-in dependency checks
	if db != nil {
		hc.RegisterDependency("database", hc.checkDatabase)
	}
	if rabbitmq != nil {
		hc.RegisterDependency("rabbitmq", hc.checkRabbitMQ)
	}

	return hc
}

// RegisterDependency registers a custom dependency check
func (hc *HealthChecker) RegisterDependency(name string, checkFunc func(context.Context) error) {
	hc.dependencies[name] = checkFunc
}

// checkDatabase checks if database connection is healthy
func (hc *HealthChecker) checkDatabase(ctx context.Context) error {
	if hc.db == nil {
		return nil
	}

	sqlDB, err := hc.db.DB()
	if err != nil {
		return err
	}

	// Ping with timeout
	return sqlDB.PingContext(ctx)
}

// checkRabbitMQ checks if RabbitMQ connection is healthy
func (hc *HealthChecker) checkRabbitMQ(ctx context.Context) error {
	if hc.rabbitmq == nil {
		return nil
	}

	if hc.rabbitmq.IsClosed() {
		return errors.New("rabbitmq connection closed")
	}

	return nil
}

// Check performs health check on all dependencies
func (hc *HealthChecker) Check(ctx context.Context) HealthStatus {
	status := HealthStatus{
		Status:       "healthy",
		Timestamp:    time.Now(),
		Dependencies: make(map[string]string),
	}

	// Check each dependency with timeout
	for name, checkFunc := range hc.dependencies {
		checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := checkFunc(checkCtx)
		cancel()

		if err != nil {
			status.Dependencies[name] = "unhealthy: " + err.Error()
			// Mark as unhealthy if any dependency fails
			status.Status = "unhealthy"
		} else {
			status.Dependencies[name] = "healthy"
		}
	}

	return status
}

// Handler returns an HTTP handler for health checks
func (hc *HealthChecker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := hlog.FromRequest(r)
		ctx := r.Context()

		status := hc.Check(ctx)

		w.Header().Set("Content-Type", "application/json")
		
		// Set appropriate HTTP status code
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

// ReadinessHandler returns a simple readiness check handler
func (hc *HealthChecker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		status := hc.Check(ctx)

		if status.Status == "healthy" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("not ready"))
		}
	}
}

// LivenessHandler returns a simple liveness check handler
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Liveness is simple - if we can respond, we're alive
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("alive"))
	}
}
