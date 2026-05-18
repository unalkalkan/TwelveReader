package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestV1PrefixRouting verifies that mounting the v1 sub-mux with a trailing-slash
// prefix pattern dispatches all /api/v1/... paths to the sub-mux handlers instead
// of returning top-level 404. Regression test for the original mount:
//   mux.Handle("/api/v1", ...) — only matched exactly "/api/v1"
// Fixed to:
//   mux.Handle("/api/v1/", ...) — prefix match for all /api/v1/... paths

func TestV1PrefixRouting(t *testing.T) {
	// Construct a v1 sub-mux mirroring the actual route registration patterns.
	v1Mux := http.NewServeMux()

	handlersHit := make(map[string]bool)

	registerHandler := func(pattern string, name string) {
		v1Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			handlersHit[name] = true
			w.WriteHeader(http.StatusOK)
		})
	}

	// Register handlers matching the actual main.go v1Mux registrations.
	registerHandler("/api/v1/health", "health")
	registerHandler("/api/v1/server-info", "server-info")
	registerHandler("/api/v1/features", "features")
	registerHandler("/api/v1/info", "info")
	registerHandler("/api/v1/providers", "providers")
	registerHandler("/api/v1/voices", "voices")
	registerHandler("/api/v1/books", "books")
	registerHandler("/api/v1/debug/events", "debug-events")

	// Wrap v1Mux in a pass-through middleware (same shape as reqCtx.Middleware).
	middleware := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		})
	}

	// Top-level mux — mirrors the fixed mount pattern in main.go.
	topMux := http.NewServeMux()
	topMux.Handle("/api/v1/", middleware(v1Mux))

	// Paths that MUST hit their v1 sub-mux handler (not 404 from top mux).
	testPaths := []struct {
		path         string
		expectedName string
	}{
		{"/api/v1/health", "health"},
		{"/api/v1/server-info", "server-info"},
		{"/api/v1/features", "features"},
		{"/api/v1/info", "info"},
		{"/api/v1/providers", "providers"},
		{"/api/v1/voices", "voices"},
		{"/api/v1/books", "books"},
		{"/api/v1/debug/events", "debug-events"},
	}

	for _, tc := range testPaths {
		t.Run(tc.path, func(t *testing.T) {
			handlersHit = make(map[string]bool)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			topMux.ServeHTTP(rec, req)

			if !handlersHit[tc.expectedName] {
				t.Errorf("path %q did not reach expected handler %q; status=%d",
					tc.path, tc.expectedName, rec.Code)
			}
			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200 for %q, got %d", tc.path, rec.Code)
			}
		})
	}

	// Verify that paths outside /api/v1/ still get 404 from the top-level mux.
	t.Run("non-v1-path-should-404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/foo", nil)
		rec := httptest.NewRecorder()
		topMux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 for /api/v2/foo, got %d", rec.Code)
		}
	})
}

// TestV1PrefixRoutingRegression demonstrates the broken behavior: using "/api/v1"
// (no trailing slash) as the mount pattern only matches the exact path, not sub-paths.
func TestV1PrefixRoutingBrokenPattern(t *testing.T) {
	v1Mux := http.NewServeMux()
	handlersHit := false

	v1Mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		handlersHit = true
		w.WriteHeader(http.StatusOK)
	})

	topMux := http.NewServeMux()
	// BROKEN: exact match — only dispatches "/api/v1", not "/api/v1/health"
	topMux.Handle("/api/v1", v1Mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	topMux.ServeHTTP(rec, req)

	// This demonstrates the bug: status is 404 because "/api/v1" exact pattern
	// does NOT match "/api/v1/health".
	if rec.Code == http.StatusOK && handlersHit {
		t.Error("unexpected: /api/v1/health reached handler with broken mount pattern")
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for broken pattern, got %d", rec.Code)
	}

	// Confirm the exact path still works (that's the limitation).
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1", nil)
	rec2 := httptest.NewRecorder()
	topMux.ServeHTTP(rec2, req2)
	// Exact match does dispatch to the sub-mux, but the sub-mux has no "/" route → 404.
	if rec2.Code == http.StatusOK && handlersHit {
		t.Error("unexpected: /api/v1 exact path hit handler")
	}
}
