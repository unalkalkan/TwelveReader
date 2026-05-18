package features

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewStore(t *testing.T) {
	initialFlags := map[string]bool{
		"feature_a": true,
		"feature_b": false,
	}
	s := NewStore(initialFlags)

	if !s.Enabled("feature_a") {
		t.Error("expected feature_a to be enabled")
	}
	if s.Enabled("feature_b") {
		t.Error("expected feature_b to be disabled")
	}
}

func TestSet(t *testing.T) {
	s := NewStore(nil)

	s.Set("new_feature", true)
	if !s.Enabled("new_feature") {
		t.Error("expected new_feature to be enabled after Set")
	}

	s.Set("new_feature", false)
	if s.Enabled("new_feature") {
		t.Error("expected new_feature to be disabled after Set(false)")
	}
}

func TestEnabledUnknownFlag(t *testing.T) {
	s := NewStore(nil)
	if s.Enabled("nonexistent") {
		t.Error("expected unknown flag to return false")
	}
}

func TestGetAll(t *testing.T) {
	initialFlags := map[string]bool{
		"x": true,
		"y": false,
	}
	s := NewStore(initialFlags)

	flags := s.GetAll()
	if len(flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(flags))
	}
}

func TestHTTPHandler(t *testing.T) {
	initialFlags := map[string]bool{
		"alpha": true,
		"beta":  false,
	}
	s := NewStore(initialFlags)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/features", nil)
	rr := httptest.NewRecorder()

	handler := s.HTTPHandler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Features["alpha"].Enabled {
		t.Error("expected alpha to be enabled in response")
	}
	if resp.Features["beta"].Enabled {
		t.Error("expected beta to be disabled in response")
	}
}
