package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
)

type requestIDKey struct{}

// RequestIDHeader is the header name for request IDs
const RequestIDHeader = "X-Request-ID"

// GenerateRequestID generates a random 16-byte hex request ID
func GenerateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use a simple counter-like value (extremely rare)
		return "err-fallback-000000000000"
	}
	return hex.EncodeToString(b)
}

// FromContext extracts the request ID from the context
func FromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}

// RequestContext adds a request ID to the request context and response header
type RequestContext struct{}

// Middleware returns an HTTP middleware that generates and attaches a request ID
func (rc *RequestContext) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(RequestIDHeader)
		if reqID == "" {
			reqID = GenerateRequestID()
		}
		w.Header().Set(RequestIDHeader, reqID)
		ctx := context.WithValue(r.Context(), requestIDKey{}, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LogWithRequestID wraps the standard log.Printf to include the request ID
func LogWithRequestID(ctx context.Context, format string, v ...interface{}) {
	reqID := FromContext(ctx)
	if reqID != "" {
		log.Printf("["+reqID+"] "+format, v...)
	} else {
		log.Printf(format, v...)
	}
}
