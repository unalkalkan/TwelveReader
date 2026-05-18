package api

import (
	"log"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
// If WriteHeader is never called, it reports 200 (the Go default).
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// NewResponseWriter creates a wrapped ResponseWriter that captures the status code.
func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

// WriteHeader captures the status code before delegating to the underlying writer.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// StatusCode returns the captured status code.
func (rw *responseWriter) StatusCode() int {
	return rw.statusCode
}

// AccessLogMiddleware returns an HTTP middleware that logs method, path, request_id,
// status_code, and duration for every request. It assumes the RequestContext middleware
// has already been applied further down the chain so the request ID is available in the
// response header (RequestContext sets X-Request-ID before calling next).
// This middleware does NOT log request bodies, headers, or secrets.
func AccessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := NewResponseWriter(w)
		next.ServeHTTP(rw, r)

		elapsed := time.Since(start)
		// RequestContext sets X-Request-ID in the response header before calling next.
		// Read it from there so we capture both client-provided and auto-generated IDs.
		reqID := w.Header().Get(RequestIDHeader)
		if reqID == "" {
			reqID = "-"
		}

		log.Printf("[ACCESS] method=%s path=%s request_id=%s status=%d duration=%s",
			r.Method, r.URL.Path, reqID, rw.statusCode, elapsed.Round(time.Millisecond))
	})
}
