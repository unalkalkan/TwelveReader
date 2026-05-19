package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware_AllowOrigin(t *testing.T) {
	cors := NewCORS()

	handler := cors.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/server-info", nil)
	req.Header.Set("Origin", "http://localhost:19002")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:19002" {
		t.Errorf("Allow-Origin = %q; want %q", got, "http://localhost:19002")
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Allow-Methods header missing")
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("Allow-Headers header missing")
	}
	if code := w.Code; code != http.StatusOK {
		t.Errorf("status = %d; want %d", code, http.StatusOK)
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	cors := NewCORS()

	handler := cors.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for OPTIONS preflight")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/server-info", nil)
	req.Header.Set("Origin", "http://localhost:19002")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if code := w.Code; code != http.StatusNoContent {
		t.Errorf("preflight status = %d; want %d", code, http.StatusNoContent)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:19002" {
		t.Errorf("Allow-Origin = %q; want %q", got, "http://localhost:19002")
	}
	if got := w.Header().Get("Access-Control-Max-Age"); got == "" {
		t.Error("Max-Age header missing on preflight response")
	}
}

func TestCORSMiddleware_NoOrigin(t *testing.T) {
	cors := NewCORS()

	handler := cors.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/server-info", nil)
	// No Origin header — same-origin request
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q; want empty (no origin sent)", got)
	}
	if code := w.Code; code != http.StatusOK {
		t.Errorf("status = %d; want %d", code, http.StatusOK)
	}
}

func TestCORSMiddleware_WildcardOrigin(t *testing.T) {
	cors := NewCORS()

	handler := cors.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/server-info", nil)
	req.Header.Set("Origin", "https://app.twelvereader.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.twelvereader.com" {
		t.Errorf("Allow-Origin = %q; want %q", got, "https://app.twelvereader.com")
	}
}
