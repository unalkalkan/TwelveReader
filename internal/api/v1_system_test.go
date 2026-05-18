package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/unalkalkan/TwelveReader/internal/features"
	"github.com/unalkalkan/TwelveReader/internal/health"
	"github.com/unalkalkan/TwelveReader/internal/provider"
)

func newTestV1SystemHandler(t *testing.T) *V1SystemHandler {
	t.Helper()
	hh := health.NewHandler("test-version")
	reg := provider.NewRegistry()
	fs := features.NewStore(map[string]bool{
		"test_feature": true,
	})
	return NewV1SystemHandler(
		hh, reg, fs,
		"test-version", "dev",
		"local", 4,
	)
}

func TestServerInfoHandler(t *testing.T) {
	h := newTestV1SystemHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/server-info", nil)
	rr := httptest.NewRecorder()

	handler := h.ServerInfoHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp ServerInfoResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Version != "test-version" {
		t.Errorf("expected version 'test-version', got %q", resp.Version)
	}
	if resp.Environment != "dev" {
		t.Errorf("expected environment 'dev', got %q", resp.Environment)
	}
	if resp.StorageAdapter != "local" {
		t.Errorf("expected storage_adapter 'local', got %q", resp.StorageAdapter)
	}
	if resp.PipelineWorkers != 4 {
		t.Errorf("expected pipeline_workers 4, got %d", resp.PipelineWorkers)
	}
	if !resp.FeatureFlags["test_feature"] {
		t.Error("expected test_feature to be enabled")
	}
	if rr.Header().Get("Content-Type") == "" {
		t.Error("expected Content-Type header")
	}
}

func TestServerInfoHandlerUptime(t *testing.T) {
	h := newTestV1SystemHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/server-info", nil)
	rr := httptest.NewRecorder()

	handler := h.ServerInfoHandler()
	handler.ServeHTTP(rr, req)

	var resp ServerInfoResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp.UptimeSeconds < 0 {
		t.Errorf("expected non-negative uptime, got %f", resp.UptimeSeconds)
	}
}

func TestFeaturesHandler(t *testing.T) {
	h := newTestV1SystemHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/features", nil)
	rr := httptest.NewRecorder()

	handler := h.FeaturesHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected application/json content type, got %s", contentType)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	features, ok := resp["features"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'features' key in response")
	}

	testFeature, ok := features["test_feature"].(map[string]interface{})
	if !ok {
		t.Fatal("expected test_feature in features")
	}

	enabled, _ := testFeature["enabled"].(bool)
	if !enabled {
		t.Error("expected test_feature to be enabled")
	}
}

func TestHealthHandler(t *testing.T) {
	hh := health.NewHandler("test-version")
	reg := provider.NewRegistry()
	fs := features.NewStore(nil)

	h := NewV1SystemHandler(
		hh, reg, fs,
		"test-version", "local",
		"local", 4,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	handler := h.HealthHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected application/json content type, got %s", contentType)
	}
}

func TestHealthHandlerWithUnhealthyCheck(t *testing.T) {
	hh := health.NewHandler("test-version")
	hh.Register("always_down", func(ctx context.Context) (health.Status, error) {
		return health.StatusUnhealthy, nil
	})
	reg := provider.NewRegistry()
	fs := features.NewStore(nil)

	h := NewV1SystemHandler(
		hh, reg, fs,
		"test-version", "local",
		"local", 4,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	handler := h.HealthHandler()
	handler.ServeHTTP(rr, req)

	// With an unhealthy check the handler returns 503
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 for unhealthy, got %d", rr.Code)
	}
}

func TestFeaturesHandlerNonGET(t *testing.T) {
	h := newTestV1SystemHandler(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/features", nil)
			rr := httptest.NewRecorder()

			handler := h.FeaturesHandler()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status 405 for %s, got %d", method, rr.Code)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			errorBody, ok := resp["error"].(map[string]interface{})
			if !ok {
				t.Fatal("expected 'error' key in response")
			}
			if errorBody["code"] != "METHOD_NOT_ALLOWED" {
				t.Errorf("expected error code METHOD_NOT_ALLOWED, got %v", errorBody["code"])
			}
		})
	}
}

func TestHealthHandlerNonGET(t *testing.T) {
	h := newTestV1SystemHandler(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/health", nil)
			rr := httptest.NewRecorder()

			handler := h.HealthHandler()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status 405 for %s, got %d", method, rr.Code)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			errorBody, ok := resp["error"].(map[string]interface{})
			if !ok {
				t.Fatal("expected 'error' key in response")
			}
			if errorBody["code"] != "METHOD_NOT_ALLOWED" {
				t.Errorf("expected error code METHOD_NOT_ALLOWED, got %v", errorBody["code"])
			}
		})
	}
}

func TestServerInfoHandlerNonGET(t *testing.T) {
	h := newTestV1SystemHandler(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/server-info", nil)
			rr := httptest.NewRecorder()

			handler := h.ServerInfoHandler()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status 405 for %s, got %d", method, rr.Code)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			errorBody, ok := resp["error"].(map[string]interface{})
			if !ok {
				t.Fatal("expected 'error' key in response")
			}
			if errorBody["code"] != "METHOD_NOT_ALLOWED" {
				t.Errorf("expected error code METHOD_NOT_ALLOWED, got %v", errorBody["code"])
			}
		})
	}
}
