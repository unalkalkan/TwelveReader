package api

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAccessLogMiddlewareLogsDefault200(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	handler := AccessLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	logLine := strings.TrimSpace(buf.String())
	if !strings.Contains(logLine, "[ACCESS]") {
		t.Fatalf("expected [ACCESS] log prefix, got: %s", logLine)
	}
	if !strings.Contains(logLine, "method=GET") {
		t.Errorf("expected method=GET in log, got: %s", logLine)
	}
	if !strings.Contains(logLine, "path=/api/v1/test") {
		t.Errorf("expected path=/api/v1/test in log, got: %s", logLine)
	}
	if !strings.Contains(logLine, "status=200") {
		t.Errorf("expected status=200 in log, got: %s", logLine)
	}
	if !strings.Contains(logLine, "duration=") {
		t.Errorf("expected duration= in log, got: %s", logLine)
	}
}

func TestAccessLogMiddlewareLogs404Status(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	handler := AccessLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	logLine := strings.TrimSpace(buf.String())
	if !strings.Contains(logLine, "status=404") {
		t.Errorf("expected status=404 in log, got: %s", logLine)
	}
}

func TestAccessLogMiddlewareCapturesDuration(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	handler := AccessLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/slow", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	logLine := strings.TrimSpace(buf.String())
	if !strings.Contains(logLine, "duration=") {
		t.Errorf("expected duration= in log, got: %s", logLine)
	}
}

// TestAccessLogWithClientProvidedRequestID is the key regression test:
// a client-provided X-Request-ID must appear in the access log entry.
func TestAccessLogWithClientProvidedRequestID(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	clientReqID := "client-provided-request-id-12345"

	// Compose: AccessLog wraps RequestContext so request ID is in context
	rc := &RequestContext{}
	handler := AccessLogMiddleware(rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set(RequestIDHeader, clientReqID)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	logLine := strings.TrimSpace(buf.String())
	if !strings.Contains(logLine, clientReqID) {
		t.Errorf("expected log to contain client request ID %q, got: %s", clientReqID, logLine)
	}
	// Also verify the response header still has it
	respReqID := rw.Header().Get(RequestIDHeader)
	if respReqID != clientReqID {
		t.Errorf("expected X-Request-ID header to be %q, got %q", clientReqID, respReqID)
	}
}

func TestAccessLogMiddlewareLogsGeneratedRequestID(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	rc := &RequestContext{}
	handler := AccessLogMiddleware(rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	// No X-Request-ID header — should be auto-generated
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	logLine := strings.TrimSpace(buf.String())
	if !strings.Contains(logLine, "request_id=") {
		t.Errorf("expected request_id= in log, got: %s", logLine)
	}
	// Should NOT contain "-" (which means no request ID was found)
	if strings.Contains(logLine, "request_id=-") {
		t.Error("expected auto-generated request ID, got dash placeholder")
	}
}

func TestAccessLogMiddlewareLogsPlaceholderWhenNoRequestID(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	handler := AccessLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	logLine := strings.TrimSpace(buf.String())
	if !strings.Contains(logLine, "request_id=-") {
		t.Errorf("expected request_id=- placeholder when no RequestContext middleware, got: %s", logLine)
	}
}

func TestResponseWriterCapturesStatus(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := NewResponseWriter(recorder)

	rw.WriteHeader(http.StatusCreated)
	if rw.StatusCode() != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rw.StatusCode())
	}

	// Test default 200 when WriteHeader is never called
	recorder2 := httptest.NewRecorder()
	rw2 := NewResponseWriter(recorder2)
	rw2.Write([]byte("ok"))
	if rw2.StatusCode() != http.StatusOK {
		t.Errorf("expected default status 200, got %d", rw2.StatusCode())
	}
}

func TestAccessLogDoesNotLogSecrets(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	rc := &RequestContext{}
	handler := AccessLogMiddleware(rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", strings.NewReader("file content"))
	req.Header.Set(RequestIDHeader, "test-id")
	req.Header.Set("Authorization", "Bearer secret-api-key-12345")
	req.Header.Set("X-Secret-Token", "super-secret-token")
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	logLine := buf.String()
	if strings.Contains(logLine, "secret-api-key-12345") {
		t.Error("access log should NOT contain API key from Authorization header")
	}
	if strings.Contains(logLine, "super-secret-token") {
		t.Error("access log should NOT contain secret tokens")
	}
	if strings.Contains(logLine, "file content") {
		t.Error("access log should NOT contain request body content")
	}
}

func TestAccessLogMultipleMethods(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	rc := &RequestContext{}
	handler := AccessLogMiddleware(rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		buf.Reset()
		req := httptest.NewRequest(method, "/api/v1/resource", nil)
		rw := httptest.NewRecorder()
		handler.ServeHTTP(rw, req)

		logLine := buf.String()
		expectedMethod := "method=" + method
		if !strings.Contains(logLine, expectedMethod) {
			t.Errorf("expected %s in log for %s request, got: %s", expectedMethod, method, logLine)
		}
	}
}
