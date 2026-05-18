package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/identity"
	_ "modernc.org/sqlite"
)

func newTestAuthHandler(t *testing.T) (*AuthHandler, *identity.AuthService, *identity.DBPool) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_identity.db")
	pool, err := identity.NewDBPool(dbPath)
	if err != nil {
		t.Fatalf("NewDBPool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	sender := &identity.LogEmailSender{}
	svc := identity.NewAuthService(pool, sender, "http://localhost:3000", "noreply@example.com",
		24*time.Hour, 7*24*time.Hour, 15*time.Minute)
	handler := NewAuthHandler(svc)
	return handler, svc, pool
}

func TestRequestMagicLink_ValidEmail(t *testing.T) {
	handler, _, _ := newTestAuthHandler(t)

	body := `{"email":"alice@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.RequestMagicLink(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp["message"], "alice@example.com") {
		t.Fatalf("response: %s", w.Body.String())
	}
}

func TestRequestMagicLink_MissingEmail(t *testing.T) {
	handler, _, _ := newTestAuthHandler(t)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.RequestMagicLink(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestRequestMagicLink_InvalidEmail(t *testing.T) {
	handler, _, _ := newTestAuthHandler(t)

	body := `{"email":"not-an-email"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.RequestMagicLink(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestRequestMagicLink_WrongMethod(t *testing.T) {
	handler, _, _ := newTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/request", nil)
	w := httptest.NewRecorder()

	handler.RequestMagicLink(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestVerifyMagicLink_ValidToken(t *testing.T) {
	handler, svc, _ := newTestAuthHandler(t)

	// Request a magic link first
	rawToken, err := svc.RequestMagicLink(context.Background(), "bob@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify?token="+rawToken, nil)
	w := httptest.NewRecorder()

	handler.VerifyMagicLink(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if _, ok := resp["session_token"]; !ok {
		t.Fatal("missing session_token in response")
	}
	if _, ok := resp["refresh_token"]; !ok {
		t.Fatal("missing refresh_token in response")
	}
	if _, ok := resp["user"]; !ok {
		t.Fatal("missing user in response")
	}
}

func TestVerifyMagicLink_MissingToken(t *testing.T) {
	handler, _, _ := newTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify", nil)
	w := httptest.NewRecorder()

	handler.VerifyMagicLink(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestVerifyMagicLink_InvalidToken(t *testing.T) {
	handler, _, _ := newTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify?token=invalid-token-here", nil)
	w := httptest.NewRecorder()

	handler.VerifyMagicLink(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body: %s", w.Code, w.Body.String())
	}
}

func TestVerifyMagicLink_AlreadyUsed(t *testing.T) {
	handler, svc, _ := newTestAuthHandler(t)

	rawToken, err := svc.RequestMagicLink(context.Background(), "carol@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}

	// Use the token once
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify?token="+rawToken, nil)
	w1 := httptest.NewRecorder()
	handler.VerifyMagicLink(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first verify: status = %d, want 200", w1.Code)
	}

	// Try to use again
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify?token="+rawToken, nil)
	w2 := httptest.NewRecorder()
	handler.VerifyMagicLink(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("second verify: status = %d, want 401", w2.Code)
	}
}

func TestRefreshSession_ValidToken(t *testing.T) {
	handler, svc, _ := newTestAuthHandler(t)

	// Login first
	rawToken, err := svc.RequestMagicLink(context.Background(), "dave@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	body := json.RawMessage(`{"refresh_token":"` + authResult.RefreshToken + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.RefreshSession(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if _, ok := resp["session_token"]; !ok {
		t.Fatal("missing session_token in response")
	}
	if _, ok := resp["refresh_token"]; !ok {
		t.Fatal("missing refresh_token in response")
	}
}

func TestRefreshSession_InvalidToken(t *testing.T) {
	handler, _, _ := newTestAuthHandler(t)

	body := json.RawMessage(`{"refresh_token":"totally-invalid"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.RefreshSession(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestLogout_WithAuthMiddleware(t *testing.T) {
	handler, svc, _ := newTestAuthHandler(t)

	// Login first
	rawToken, err := svc.RequestMagicLink(context.Background(), "eve@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Create a middleware-wrapped handler
	middleware := SessionAuthMiddleware(svc)
	wrapped := middleware(http.HandlerFunc(handler.Logout))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+authResult.SessionToken)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["message"] != "logged out successfully" {
		t.Fatalf("unexpected message: %s", resp["message"])
	}
}

func TestLogout_NoSession(t *testing.T) {
	handler, _, _ := newTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()

	handler.Logout(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestMe_WithAuthMiddleware(t *testing.T) {
	handler, svc, _ := newTestAuthHandler(t)

	// Login first
	rawToken, err := svc.RequestMagicLink(context.Background(), "frank@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	middleware := SessionAuthMiddleware(svc)
	wrapped := middleware(http.HandlerFunc(handler.Me))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+authResult.SessionToken)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	user := resp["user"].(map[string]interface{})
	if user["email"] != "frank@example.com" {
		t.Fatalf("unexpected email: %v", user["email"])
	}
}

func TestMe_NoSession(t *testing.T) {
	handler, _, _ := newTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()

	handler.Me(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestRequireAuth_NoToken(t *testing.T) {
	handler, svc, _ := newTestAuthHandler(t)

	middleware := RequireAuth(svc)
	wrapped := middleware(http.HandlerFunc(handler.Me))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	handler, svc, _ := newTestAuthHandler(t)

	middleware := RequireAuth(svc)
	wrapped := middleware(http.HandlerFunc(handler.Me))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-value")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestFullAuthFlow_HTTP(t *testing.T) {
	/*
		Test the complete flow through HTTP handlers:
		1. POST /auth/request -> magic link sent
		2. GET /auth/verify?token=X -> session + refresh tokens
		3. GET /auth/me (with Bearer token) -> user info
		4. POST /auth/refresh -> new tokens
		5. POST /auth/logout -> logged out
		6. GET /auth/me (old token) -> 401
	*/
	handler, svc, _ := newTestAuthHandler(t)

	middleware := SessionAuthMiddleware(svc)
	requireMiddleware := RequireAuth(svc)

	meHandler := requireMiddleware(http.HandlerFunc(handler.Me))
	logoutHandler := middleware(http.HandlerFunc(handler.Logout))
	refreshHandler := middleware(http.HandlerFunc(handler.RefreshSession))

	// Step 1: Request magic link
	body1 := `{"email":"flow@example.com"}`
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/request", strings.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.RequestMagicLink(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("step 1: status = %d", w1.Code)
	}

	// Get the raw token from the service (since we use LogEmailSender)
	rawToken, err := svc.RequestMagicLink(context.Background(), "flow2@example.com")
	if err != nil {
		t.Fatalf("get raw token: %v", err)
	}

	// Step 2: Verify magic link
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify?token="+rawToken, nil)
	w2 := httptest.NewRecorder()
	handler.VerifyMagicLink(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("step 2: status = %d, body: %s", w2.Code, w2.Body.String())
	}

	var authResp map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&authResp)
	sessionToken := authResp["session_token"].(string)
	refreshToken := authResp["refresh_token"].(string)

	// Step 3: GET /me
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req3.Header.Set("Authorization", "Bearer "+sessionToken)
	w3 := httptest.NewRecorder()
	meHandler.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("step 3: status = %d", w3.Code)
	}

	var meResp map[string]interface{}
	json.NewDecoder(w3.Body).Decode(&meResp)
	user := meResp["user"].(map[string]interface{})
	if user["email"] != "flow2@example.com" {
		t.Fatalf("step 3: unexpected email: %v", user["email"])
	}

	// Step 4: Refresh session
	body4 := json.RawMessage(`{"refresh_token":"` + refreshToken + `"}`)
	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body4))
	req4.Header.Set("Content-Type", "application/json")
	req4.Header.Set("Authorization", "Bearer "+sessionToken)
	w4 := httptest.NewRecorder()
	refreshHandler.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Fatalf("step 4: status = %d, body: %s", w4.Code, w4.Body.String())
	}

	var refreshResp map[string]interface{}
	json.NewDecoder(w4.Body).Decode(&refreshResp)
	newSessionToken := refreshResp["session_token"].(string)
	if newSessionToken == sessionToken {
		t.Fatal("step 4: session token should change on refresh")
	}

	// Step 5: Logout
	req5 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req5.Header.Set("Authorization", "Bearer "+sessionToken)
	w5 := httptest.NewRecorder()
	logoutHandler.ServeHTTP(w5, req5)
	if w5.Code != http.StatusOK {
		t.Fatalf("step 5: status = %d", w5.Code)
	}

	// Step 6: Try to access /me with old (logged-out) token -> should fail
	req6 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req6.Header.Set("Authorization", "Bearer "+sessionToken)
	w6 := httptest.NewRecorder()
	meHandler.ServeHTTP(w6, req6)
	if w6.Code != http.StatusUnauthorized {
		t.Fatalf("step 6: status = %d, want 401", w6.Code)
	}

	// Step 7: New session token (from refresh) is still valid - logout only revokes the specific session passed
	req7 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req7.Header.Set("Authorization", "Bearer "+newSessionToken)
	w7 := httptest.NewRecorder()
	meHandler.ServeHTTP(w7, req7)
	if w7.Code != http.StatusOK {
		t.Fatalf("step 7: status = %d, want 200 (refreshed session should still be valid)", w7.Code)
	}
}
