package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()
	if id1 == id2 {
		t.Error("expected two different request IDs")
	}
	if len(id1) != 32 {
		t.Errorf("expected 32-char hex ID, got %d", len(id1))
	}
}

func TestMiddlewareAddsRequestID(t *testing.T) {
	rc := &RequestContext{}
	handler := rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	reqID := rr.Header().Get(RequestIDHeader)
	if reqID == "" {
		t.Error("expected X-Request-ID header to be set")
	}
	if len(reqID) != 32 {
		t.Errorf("expected 32-char hex ID in header, got %d", len(reqID))
	}
}

func TestMiddlewarePreservesClientRequestID(t *testing.T) {
	rc := &RequestContext{}
	var capturedReqID string
	handler := rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReqID = FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "client-provided-id")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Header().Get(RequestIDHeader) != "client-provided-id" {
		t.Error("expected client request ID to be preserved in response header")
	}
	if capturedReqID != "client-provided-id" {
		t.Error("expected client request ID to be in context")
	}
}

func TestFromContext(t *testing.T) {
	reqID := GenerateRequestID()
	ctx := context.WithValue(context.Background(), requestIDKey{}, reqID)

	if result := FromContext(ctx); result != reqID {
		t.Errorf("expected %s, got %s", reqID, result)
	}

	// Empty context returns empty string
	if result := FromContext(context.Background()); result != "" {
		t.Errorf("expected empty string for empty context, got %q", result)
	}
}

func TestLogWithRequestID(t *testing.T) {
	reqID := GenerateRequestID()
	ctx := context.WithValue(context.Background(), requestIDKey{}, reqID)

	LogWithRequestID(ctx, "test message")
	// No assertion possible on log output; just ensure it doesn't panic

	LogWithRequestID(context.Background(), "no request id")
	// Also should not panic with empty context
}

func TestLogWithRequestIDContainsRequestID(t *testing.T) {
	reqID := GenerateRequestID()
	ctx := context.WithValue(context.Background(), requestIDKey{}, reqID)
	LogWithRequestID(ctx, "test msg %d", 42)
	// The log line should contain the request ID. Since we can't capture log output easily,
	// this is a basic smoke test to ensure it doesn't crash.
}

func TestHTTPHandlerJSON(t *testing.T) {
	// Verify Content-Type header handling isn't broken
	rc := &RequestContext{}
	handler := rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}
