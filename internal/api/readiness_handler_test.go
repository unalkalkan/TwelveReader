package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/features"
	"github.com/unalkalkan/TwelveReader/internal/health"
	"github.com/unalkalkan/TwelveReader/internal/provider"
)

func newTestReadinessHandler(t *testing.T) *ReadinessHandler {
	t.Helper()
	hh := health.NewHandler("test-version")
	reg := provider.NewRegistry()
	fs := features.NewStore(map[string]bool{
		"saas_auth":      false,
		"usage_metering": true,
	})

	v1System := NewV1SystemHandler(
		hh, reg, fs,
		"test-version", "dev",
		"local", 4,
	)

	return NewReadinessHandler(v1System, hh, fs, "test-version", "dev")
}

func TestReadinessSmoke_OK(t *testing.T) {
	h := newTestReadinessHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/debug/readiness/smoke", nil)
	rr := httptest.NewRecorder()

	h.Smoke(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp SmokeVisibilityResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v, body: %s", err, rr.Body.String())
	}

	if resp.Overall != "all_ok" {
		t.Errorf("expected overall 'all_ok', got %q", resp.Overall)
	}

	if len(resp.Checks) != 3 {
		t.Fatalf("expected 3 checks, got %d", len(resp.Checks))
	}

	// Verify all three endpoints are present
	checkNames := make(map[string]bool)
	for _, check := range resp.Checks {
		checkNames[check.Name] = true
		if check.Status != "ok" {
			t.Errorf("check %q: expected status 'ok', got %q", check.Name, check.Status)
		}
		if check.HttpCode != 200 {
			t.Errorf("check %q: expected http_code 200, got %d", check.Name, check.HttpCode)
		}
		if check.LatencyMs < 0 {
			t.Errorf("check %q: expected non-negative latency, got %f", check.Name, check.LatencyMs)
		}
	}

	expectedChecks := map[string]bool{"health": true, "server-info": true, "features": true}
	if len(checkNames) != len(expectedChecks) {
		t.Errorf("expected checks %v, got %v", expectedChecks, checkNames)
	}

	// Verify timestamp is recent
	if time.Since(resp.Timestamp) > 5*time.Second {
		t.Errorf("timestamp seems stale: %v", resp.Timestamp)
	}
}

func TestReadinessSmoke_PathPresent(t *testing.T) {
	h := newTestReadinessHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/debug/readiness/smoke", nil)
	rr := httptest.NewRecorder()

	h.Smoke(rr, req)

	var resp SmokeVisibilityResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	expectedPaths := map[string]string{
		"health":      "/api/v1/health",
		"server-info": "/api/v1/server-info",
		"features":    "/api/v1/features",
	}

	for _, check := range resp.Checks {
		expectedPath, ok := expectedPaths[check.Name]
		if !ok {
			t.Errorf("unexpected check name: %q", check.Name)
			continue
		}
		if check.Path != expectedPath {
			t.Errorf("check %q: expected path %q, got %q", check.Name, expectedPath, check.Path)
		}
	}
}

func TestReadinessSmoke_MethodNotAllowed(t *testing.T) {
	h := newTestReadinessHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/debug/readiness/smoke", nil)
	rr := httptest.NewRecorder()

	h.Smoke(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for POST, got %d", rr.Code)
	}
}

func TestReadinessSmoke_HealthUnhealthy(t *testing.T) {
	hh := health.NewHandler("test-version")
	hh.Register("always_down", func(ctx context.Context) (health.Status, error) {
		return health.StatusUnhealthy, nil
	})
	reg := provider.NewRegistry()
	fs := features.NewStore(nil)

	v1System := NewV1SystemHandler(hh, reg, fs, "test-version", "dev", "local", 4)
	h := NewReadinessHandler(v1System, hh, fs, "test-version", "dev")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/debug/readiness/smoke", nil)
	rr := httptest.NewRecorder()

	h.Smoke(rr, req)

	var resp SmokeVisibilityResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Health check should be in error state
	healthCheck := findCheck(resp.Checks, "health")
	if healthCheck == nil {
		t.Fatal("health check not found in response")
	}

	if healthCheck.Status != "error" {
		t.Errorf("expected health check status 'error', got %q", healthCheck.Status)
	}

	if healthCheck.HttpCode != 503 {
		t.Errorf("expected health check http_code 503, got %d", healthCheck.HttpCode)
	}

	if healthCheck.Error == "" {
		t.Error("expected health check to have an error message")
	}

	// Overall should be unhealthy due to the unhealthy health check
	if resp.Overall != "unhealthy" {
		t.Errorf("expected overall 'unhealthy', got %q", resp.Overall)
	}
}

func TestReadinessSmoke_HealthDegraded(t *testing.T) {
	hh := health.NewHandler("test-version")
	hh.Register("slow_provider", func(ctx context.Context) (health.Status, error) {
		return health.StatusDegraded, nil
	})
	reg := provider.NewRegistry()
	fs := features.NewStore(nil)

	v1System := NewV1SystemHandler(hh, reg, fs, "test-version", "dev", "local", 4)
	h := NewReadinessHandler(v1System, hh, fs, "test-version", "dev")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/debug/readiness/smoke", nil)
	rr := httptest.NewRecorder()

	h.Smoke(rr, req)

	var resp SmokeVisibilityResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	healthCheck := findCheck(resp.Checks, "health")
	if healthCheck == nil {
		t.Fatal("health check not found in response")
	}

	if healthCheck.Status != "warning" {
		t.Errorf("expected health check status 'warning', got %q", healthCheck.Status)
	}

	// Overall should be degraded (not unhealthy, since degraded is a warning)
	if resp.Overall != "degraded" {
		t.Errorf("expected overall 'degraded', got %q", resp.Overall)
	}
}

func TestReadinessSmoke_ServerInfoDataContainsFields(t *testing.T) {
	h := newTestReadinessHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/debug/readiness/smoke", nil)
	rr := httptest.NewRecorder()

	h.Smoke(rr, req)

	var resp SmokeVisibilityResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	serverInfoCheck := findCheck(resp.Checks, "server-info")
	if serverInfoCheck == nil {
		t.Fatal("server-info check not found")
	}

	// Verify the data field contains expected server info fields
	serverInfoRaw, err := json.Marshal(serverInfoCheck.Data)
	if err != nil {
		t.Fatalf("failed to marshal server-info data: %v", err)
	}

	var serverInfo map[string]interface{}
	json.Unmarshal(serverInfoRaw, &serverInfo)

	expectedFields := []string{"version", "environment", "uptime_seconds", "storage_adapter"}
	for _, field := range expectedFields {
		if _, exists := serverInfo[field]; !exists {
			t.Errorf("expected server-info data to contain %q field", field)
		}
	}

	// Verify version matches
	if serverInfo["version"] != "test-version" {
		t.Errorf("expected server-info version 'test-version', got %v", serverInfo["version"])
	}
}

func TestReadinessSmoke_FeaturesDataContainsFlags(t *testing.T) {
	h := newTestReadinessHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/debug/readiness/smoke", nil)
	rr := httptest.NewRecorder()

	h.Smoke(rr, req)

	var resp SmokeVisibilityResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	featuresCheck := findCheck(resp.Checks, "features")
	if featuresCheck == nil {
		t.Fatal("features check not found")
	}

	featuresRaw, err := json.Marshal(featuresCheck.Data)
	if err != nil {
		t.Fatal("failed to marshal features data")
	}

	var featuresData map[string]interface{}
	json.Unmarshal(featuresRaw, &featuresData)

	// Should have a "features" key
	if _, exists := featuresData["features"]; !exists {
		t.Error("expected features data to contain 'features' key")
	}

	// Check that our test flags are present
	featuresMap, isMap := featuresData["features"].(map[string]interface{})
	if !isMap {
		t.Fatal("expected 'features' to be a map")
	}

	// saas_auth should be false, usage_metering should be true
	saasAuth, hasSaasAuth := featuresMap["saas_auth"]
	if !hasSaasAuth {
		t.Error("expected saas_auth flag in features data")
	} else {
		flagData, _ := saasAuth.(map[string]interface{})
		if flagData != nil && flagData["enabled"] != false {
			t.Error("expected saas_auth to be disabled")
		}
	}

	usageMetering, hasUsageMetering := featuresMap["usage_metering"]
	if !hasUsageMetering {
		t.Error("expected usage_metering flag in features data")
	} else {
		flagData, _ := usageMetering.(map[string]interface{})
		if flagData != nil && flagData["enabled"] != true {
			t.Error("expected usage_metering to be enabled")
		}
	}
}

func TestSmokeCheckResultJSON(t *testing.T) {
	check := SmokeCheckResult{
		Name:      "test",
		Path:      "/api/v1/test",
		Status:    "ok",
		HttpCode:  200,
		LatencyMs: 1.5,
	}

	data, err := json.Marshal(check)
	if err != nil {
		t.Fatalf("failed to marshal SmokeCheckResult: %v", err)
	}

	var decoded SmokeCheckResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal SmokeCheckResult: %v", err)
	}

	if decoded.Name != check.Name || decoded.Path != check.Path || decoded.Status != check.Status {
		t.Error("JSON round-trip failed for SmokeCheckResult")
	}
}

func TestSmokeVisibilityResponseJSON(t *testing.T) {
	resp := SmokeVisibilityResponse{
		Timestamp: time.Now().UTC(),
		Checks: []SmokeCheckResult{
			{Name: "health", Path: "/api/v1/health", Status: "ok", HttpCode: 200},
			{Name: "server-info", Path: "/api/v1/server-info", Status: "ok", HttpCode: 200},
		},
		Overall: "all_ok",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal SmokeVisibilityResponse: %v", err)
	}

	var decoded SmokeVisibilityResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal SmokeVisibilityResponse: %v", err)
	}

	if decoded.Overall != "all_ok" || len(decoded.Checks) != 2 {
		t.Error("JSON round-trip failed for SmokeVisibilityResponse")
	}
}

func TestContent_TypeHeader(t *testing.T) {
	h := newTestReadinessHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/debug/readiness/smoke", nil)
	rr := httptest.NewRecorder()

	h.Smoke(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected application/json content type, got %s", contentType)
	}
}

// findCheck returns the check with the given name from the slice, or nil.
func findCheck(checks []SmokeCheckResult, name string) *SmokeCheckResult {
	for i := range checks {
		if checks[i].Name == name {
			return &checks[i]
		}
	}
	return nil
}
