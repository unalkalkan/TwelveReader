package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/features"
	"github.com/unalkalkan/TwelveReader/internal/health"
	"github.com/unalkalkan/TwelveReader/internal/provider"
)

// ServerInfoResponse represents the /api/v1/server-info response
type ServerInfoResponse struct {
	Version         string            `json:"version"`
	Environment     string            `json:"environment"`
	UptimeSeconds   float64           `json:"uptime_seconds"`
	StorageAdapter  string            `json:"storage_adapter"`
	LLMProviders    []string          `json:"llm_providers"`
	TTSProviders    []string          `json:"tts_providers"`
	OCRProviders    []string          `json:"ocr_providers"`
	PipelineWorkers int               `json:"pipeline_workers"`
	FeatureFlags    map[string]bool   `json:"feature_flags"`
}

// V1SystemHandler provides handlers for the /api/v1 system endpoints
type V1SystemHandler struct {
	healthHandler *health.Handler
	providerReg   *provider.Registry
	featureStore  *features.Store
	version       string
	environment   string
	startupTime   time.Time
	storage       string // storage adapter name
	pipelineSize  int    // worker pool size
}

// NewV1SystemHandler creates a new system handler for /api/v1 endpoints
func NewV1SystemHandler(
	hh *health.Handler,
	reg *provider.Registry,
	fs *features.Store,
	version, environment string,
	storageAdapter string,
	workerPoolSize int,
) *V1SystemHandler {
	return &V1SystemHandler{
		healthHandler: hh,
		providerReg:   reg,
		featureStore:  fs,
		version:       version,
		environment:   environment,
		startupTime:   time.Now(),
		storage:       storageAdapter,
		pipelineSize:  workerPoolSize,
	}
}

// HealthHandler returns an HTTP handler for GET /api/v1/health
func (h *V1SystemHandler) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := h.healthHandler.RunChecks(r.Context())

		w.Header().Set("Content-Type", "application/json")
		statusCode := http.StatusOK
		if response.Status == health.StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	}
}

// ServerInfoHandler returns an HTTP handler for GET /api/v1/server-info
func (h *V1SystemHandler) ServerInfoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		llmProviders := h.providerReg.ListLLM()
		ttsProviders := h.providerReg.ListTTS()
		ocrProviders := h.providerReg.ListOCR()

		// Build feature flags summary (flag_name: enabled)
		flagSummary := make(map[string]bool)
		allFlags := h.featureStore.GetAll()
		for name, flag := range allFlags {
			flagSummary[name] = flag.Enabled
		}

		resp := ServerInfoResponse{
			Version:         h.version,
			Environment:     h.environment,
			UptimeSeconds:   time.Since(h.startupTime).Seconds(),
			StorageAdapter:  h.storage,
			LLMProviders:    llmProviders,
			TTSProviders:    ttsProviders,
			OCRProviders:    ocrProviders,
			PipelineWorkers: h.pipelineSize,
			FeatureFlags:    flagSummary,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

// FeaturesHandler returns an HTTP handler for GET /api/v1/features
func (h *V1SystemHandler) FeaturesHandler() http.HandlerFunc {
	return h.featureStore.HTTPHandler()
}
