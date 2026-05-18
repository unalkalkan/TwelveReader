package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/unalkalkan/TwelveReader/internal/identity"
)

// AuthHandler handles authentication-related HTTP endpoints.
type AuthHandler struct {
	authService *identity.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *identity.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// RequestBody for auth requests.
type authRequestEmail struct {
	Email string `json:"email"`
}

type authRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authVerifyQuery struct {
	Token string // from query param ?token=...
}

// RequestMagicLink handles POST /api/v1/auth/request
func (h *AuthHandler) RequestMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteMethodNotAllowedError(w, r)
		return
	}

	var req authRequestEmail
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteStructuredError(w, r, ErrCodeBadRequest, "invalid JSON body", http.StatusBadRequest)
		return
	}

	req.Email = trimSpace(req.Email)
	if req.Email == "" {
		WriteStructuredError(w, r, ErrCodeBadRequest, "email is required", http.StatusBadRequest)
		return
	}

	_, err := h.authService.RequestMagicLink(r.Context(), req.Email)
	if err != nil {
		WriteStructuredError(w, r, ErrCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Magic link sent to " + req.Email,
	})
}

// VerifyMagicLink handles GET /api/v1/auth/verify?token=...
func (h *AuthHandler) VerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteMethodNotAllowedError(w, r)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		WriteStructuredError(w, r, ErrCodeBadRequest, "token query parameter is required", http.StatusBadRequest)
		return
	}

	ipAddress := extractIP(r)
	userAgent := r.UserAgent()

	result, err := h.authService.VerifyMagicLink(r.Context(), token, ipAddress, userAgent)
	if err != nil {
		WriteStructuredError(w, r, ErrCodeUnauthorized, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":           result.User,
		"session_token":  result.SessionToken,
		"refresh_token":  result.RefreshToken,
	})
}

// RefreshSession handles POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteMethodNotAllowedError(w, r)
		return
	}

	var req authRefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteStructuredError(w, r, ErrCodeBadRequest, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.RefreshToken == "" {
		WriteStructuredError(w, r, ErrCodeBadRequest, "refresh_token is required", http.StatusBadRequest)
		return
	}

	ipAddress := extractIP(r)
	userAgent := r.UserAgent()

	result, err := h.authService.RefreshSession(r.Context(), req.RefreshToken, ipAddress, userAgent)
	if err != nil {
		WriteStructuredError(w, r, ErrCodeUnauthorized, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_token":  result.SessionToken,
		"refresh_token":  result.RefreshToken,
	})
}

// Logout handles POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteMethodNotAllowedError(w, r)
		return
	}

	sessionID := GetSessionIDFromContext(r.Context())
	if sessionID == "" {
		WriteStructuredError(w, r, ErrCodeUnauthorized, "no active session", http.StatusUnauthorized)
		return
	}

	err := h.authService.Logout(r.Context(), sessionID)
	if err != nil {
		WriteStructuredError(w, r, ErrCodeUnauthorized, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "logged out successfully",
	})
}

// Me handles GET /api/v1/auth/me - returns current authenticated user info.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteMethodNotAllowedError(w, r)
		return
	}

	user := GetUserFromContext(r.Context())
	if user == nil {
		WriteStructuredError(w, r, ErrCodeUnauthorized, "not authenticated", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": user,
	})
}

func trimSpace(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func extractIP(r *http.Request) string {
	// Check common forwarded headers first
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"} {
		if ip := r.Header.Get(header); ip != "" {
			return ip
		}
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}
