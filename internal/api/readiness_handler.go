package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/features"
	"github.com/unalkalkan/TwelveReader/internal/health"
)

// SmokeCheckResult represents the result of a single readiness endpoint smoke check.
type SmokeCheckResult struct {
	Name      string      `json:"name"`
	Path      string      `json:"path"`
	Status    string      `json:"status"`     // "ok", "warning", "error"
	HttpCode  int         `json:"http_code"`
	LatencyMs float64     `json:"latency_ms"`
	Error     string      `json:"error,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// SmokeVisibilityResponse aggregates readiness endpoint status for the debug dashboard.
type SmokeVisibilityResponse struct {
	Timestamp time.Time          `json:"timestamp"`
	Checks    []SmokeCheckResult `json:"checks"`
	Overall   string             `json:"overall"` // "all_ok", "degraded", "unhealthy"
}

// ReadinessHandler provides smoke visibility for readiness endpoints in the debug dashboard.
type ReadinessHandler struct {
	v1System      *V1SystemHandler
	healthHandler *health.Handler
	featureStore  *features.Store
	version       string
	environment   string
	startupTime   time.Time
}

// NewReadinessHandler creates a new readiness smoke visibility handler.
func NewReadinessHandler(
	v1 *V1SystemHandler,
	hh *health.Handler,
	fs *features.Store,
	version, environment string,
) *ReadinessHandler {
	return &ReadinessHandler{
		v1System:      v1,
		healthHandler: hh,
		featureStore:  fs,
		version:       version,
		environment:   environment,
		startupTime:   time.Now(),
	}
}

// Smoke checks all readiness endpoints and returns their status.
func (h *ReadinessHandler) Smoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteMethodNotAllowedError(w, r)
		return
	}

	checks := make([]SmokeCheckResult, 0, 3)
	overall := "all_ok"

	// 1. Check /api/v1/health
	check := h.checkHealth(r.Context())
	checks = append(checks, check)
	if check.Status == "error" {
		overall = "unhealthy"
	} else if check.Status == "warning" && overall == "all_ok" {
		overall = "degraded"
	}

	// 2. Check /api/v1/server-info
	check = h.checkServerInfo()
	checks = append(checks, check)
	if check.Status != "ok" && overall == "all_ok" {
		overall = "degraded"
	}

	// 3. Check /api/v1/features
	check = h.checkFeatures()
	checks = append(checks, check)
	if check.Status != "ok" && overall == "all_ok" {
		overall = "degraded"
	}

	resp := SmokeVisibilityResponse{
		Timestamp: time.Now().UTC(),
		Checks:    checks,
		Overall:   overall,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *ReadinessHandler) checkHealth(ctx context.Context) SmokeCheckResult {
	start := time.Now()

	result := SmokeCheckResult{
		Name:     "health",
		Path:     "/api/v1/health",
		HttpCode: 200,
	}

	healthResponse := h.healthHandler.RunChecks(ctx)
	result.LatencyMs = float64(time.Since(start).Microseconds()) / 1000.0

	switch healthResponse.Status {
	case health.StatusHealthy:
		result.Status = "ok"
		result.HttpCode = 200
	case health.StatusDegraded:
		result.Status = "warning"
		result.HttpCode = 200
	case health.StatusUnhealthy:
		result.Status = "error"
		result.HttpCode = 503
		// Collect errors from individual checks
		var errorParts []string
		for name, cr := range healthResponse.Checks {
			if cr.Error != "" {
				errorParts = append(errorParts, name+": "+cr.Error)
			}
		}
		result.Error = "unhealthy: " + strings.Join(errorParts, ", ")
	default:
		result.Status = "error"
		result.Error = "unknown health status: " + string(healthResponse.Status)
	}

	result.Data = healthResponse
	return result
}

func (h *ReadinessHandler) checkServerInfo() SmokeCheckResult {
	start := time.Now()

	result := SmokeCheckResult{
		Name:     "server-info",
		Path:     "/api/v1/server-info",
		HttpCode: 200,
	}

	llmProviders := h.v1System.providerReg.ListLLM()
	ttsProviders := h.v1System.providerReg.ListTTS()
	ocrProviders := h.v1System.providerReg.ListOCR()

	flagSummary := make(map[string]bool)
	allFlags := h.featureStore.GetAll()
	for name, flag := range allFlags {
		flagSummary[name] = flag.Enabled
	}

	serverInfo := ServerInfoResponse{
		Version:         h.version,
		Environment:     h.environment,
		UptimeSeconds:   time.Since(h.startupTime).Seconds(),
		StorageAdapter:  h.v1System.storage,
		LLMProviders:    llmProviders,
		TTSProviders:    ttsProviders,
		OCRProviders:    ocrProviders,
		PipelineWorkers: h.v1System.pipelineSize,
		FeatureFlags:    flagSummary,
	}

	result.LatencyMs = float64(time.Since(start).Microseconds()) / 1000.0
	result.Status = "ok"
	result.Data = serverInfo
	return result
}

func (h *ReadinessHandler) checkFeatures() SmokeCheckResult {
	start := time.Now()

	result := SmokeCheckResult{
		Name:     "features",
		Path:     "/api/v1/features",
		HttpCode: 200,
	}

	allFlags := h.featureStore.GetAll()
	featuresResponse := features.Response{Features: allFlags}

	result.LatencyMs = float64(time.Since(start).Microseconds()) / 1000.0
	result.Status = "ok"
	result.Data = featuresResponse
	return result
}
