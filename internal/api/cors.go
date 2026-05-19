package api

import (
	"net/http"
	"strings"
)

// CORS config for /api/v1 endpoints.
// Custom/self-hosted servers are validated from different origins (e.g. localhost:19002 -> server),
// so cross-origin access must be allowed.
type CORS struct {
	allowedOrigins []string
	allowedMethods []string
	allowedHeaders []string
}

// NewCORS creates a CORS middleware with sensible defaults for the TwelveReader API.
func NewCORS() *CORS {
	return &CORS{
		// Allow any origin — self-hosted servers are accessed from arbitrary client origins.
		// The server validation flow explicitly checks /api/v1/server-info to verify identity.
		allowedOrigins: []string{"*"},
		allowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		allowedHeaders: []string{
			"Content-Type",
			"Authorization",
			RequestIDHeader,
		},
	}
}

// Middleware returns an HTTP middleware that handles CORS headers.
func (c *CORS) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.allowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.allowedHeaders, ", "))
			w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
		}

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
