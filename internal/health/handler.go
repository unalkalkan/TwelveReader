package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// Check represents a health check
type Check struct {
	Name   string                                    `json:"name"`
	Status Status                                    `json:"status"`
	Error  string                                    `json:"error,omitempty"`
	Check  func(ctx context.Context) (Status, error) `json:"-"`
}

// Response represents a health check response
type Response struct {
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks,omitempty"`
	Version   string                 `json:"version,omitempty"`
}

// CheckResult represents the result of a single health check
type CheckResult struct {
	Status Status `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Handler manages health checks
type Handler struct {
	checks  map[string]*Check
	mu      sync.RWMutex
	version string
}

// NewHandler creates a new health check handler
func NewHandler(version string) *Handler {
	return &Handler{
		checks:  make(map[string]*Check),
		version: version,
	}
}

// Register adds a health check
func (h *Handler) Register(name string, checkFunc func(ctx context.Context) (Status, error)) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.checks[name] = &Check{
		Name:  name,
		Check: checkFunc,
	}
}

// RunChecks executes all registered health checks
func (h *Handler) RunChecks(ctx context.Context) Response {
	h.mu.RLock()
	checks := make(map[string]*Check, len(h.checks))
	for k, v := range h.checks {
		checks[k] = v
	}
	h.mu.RUnlock()

	results := make(map[string]CheckResult)
	overallStatus := StatusHealthy

	for name, check := range checks {
		status, err := check.Check(ctx)
		result := CheckResult{
			Status: status,
		}
		if err != nil {
			result.Error = err.Error()
		}

		results[name] = result

		// Determine overall status
		if status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if status == StatusDegraded && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}

	return Response{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Checks:    results,
		Version:   h.version,
	}
}

// LivenessHandler returns an HTTP handler for liveness checks
// Liveness checks determine if the application is running
func (h *Handler) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Status:    StatusHealthy,
			Timestamp: time.Now(),
			Version:   h.version,
		})
	}
}

// ReadinessHandler returns an HTTP handler for readiness checks
// Readiness checks determine if the application is ready to serve traffic
func (h *Handler) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		response := h.RunChecks(ctx)

		w.Header().Set("Content-Type", "application/json")

		// Return 503 if unhealthy, 200 otherwise
		statusCode := http.StatusOK
		if response.Status == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}

		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	}
}

// HealthHandler returns an HTTP handler for full health checks
func (h *Handler) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		response := h.RunChecks(ctx)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
