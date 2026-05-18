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

	sender := &identity.LogEmailSender{DevMode: true}
	svc := identity.NewAuthService(pool, sender, "http://localhost:3000", "noreply@example.com",
		24*time.Hour, 7*24*time.Hour, 15*time.Minute)
	handler := NewAuthHandler(svc, pool)
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
	// Verify role_name is present in response (client uses this to enforce UI access control)
	if _, ok := resp["role_name"].(string); !ok {
		t.Fatal("expected role_name field in /auth/me response")
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
		4. POST /auth/refresh -> new tokens, old session revoked
		5. Old session invalid after refresh
		6. New session valid
		7. POST /auth/logout (new session) -> logged out
		8. New session invalid after logout
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

	// Step 5: Old session is revoked after refresh (proper lifecycle)
	req5 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req5.Header.Set("Authorization", "Bearer "+sessionToken)
	w5 := httptest.NewRecorder()
	meHandler.ServeHTTP(w5, req5)
	if w5.Code != http.StatusUnauthorized {
		t.Fatalf("step 5: status = %d, want 401 (old session revoked by refresh)", w5.Code)
	}

	// Step 6: New session is valid
	req6 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req6.Header.Set("Authorization", "Bearer "+newSessionToken)
	w6 := httptest.NewRecorder()
	meHandler.ServeHTTP(w6, req6)
	if w6.Code != http.StatusOK {
		t.Fatalf("step 6: status = %d, want 200 (new session should be valid)", w6.Code)
	}

	// Step 7: Logout using new session token
	req7 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req7.Header.Set("Authorization", "Bearer "+newSessionToken)
	w7 := httptest.NewRecorder()
	logoutHandler.ServeHTTP(w7, req7)
	if w7.Code != http.StatusOK {
		t.Fatalf("step 7: status = %d, want 200", w7.Code)
	}

	// Step 8: New session is invalid after logout
	req8 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req8.Header.Set("Authorization", "Bearer "+newSessionToken)
	w8 := httptest.NewRecorder()
	meHandler.ServeHTTP(w8, req8)
	if w8.Code != http.StatusUnauthorized {
		t.Fatalf("step 8: status = %d, want 401 (session revoked by logout)", w8.Code)
	}
}

func TestVerifyMagicLink_ResponseCacheHeaders(t *testing.T) {
	handler, svc, _ := newTestAuthHandler(t)

	rawToken, err := svc.RequestMagicLink(context.Background(), "cache-verify@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify?token="+rawToken, nil)
	w := httptest.NewRecorder()

	handler.VerifyMagicLink(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	// Verify cache headers are present on token-bearing response
	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl == "" {
		t.Fatal("Cache-Control header missing on /auth/verify response")
	}
	if !strings.Contains(cacheControl, "no-store") {
		t.Fatalf("Cache-Control should contain 'no-store', got: %s", cacheControl)
	}
}

func TestRefreshSession_ResponseCacheHeaders(t *testing.T) {
	handler, svc, _ := newTestAuthHandler(t)

	rawToken, err := svc.RequestMagicLink(context.Background(), "cache-refresh@example.com")
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
		t.Fatalf("status = %d, want 200", w.Code)
	}

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl == "" {
		t.Fatal("Cache-Control header missing on /auth/refresh response")
	}
	if !strings.Contains(cacheControl, "no-store") {
		t.Fatalf("Cache-Control should contain 'no-store', got: %s", cacheControl)
	}
}

func TestMiddleware_RejectsSuspendedUser(t *testing.T) {
	handler, svc, pool := newTestAuthHandler(t)

	rawToken, err := svc.RequestMagicLink(context.Background(), "suspended-mw@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Suspend the user
	user, err := pool.Users.GetUserByID(context.Background(), authResult.User.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	user.Status = "suspended"
	if err := pool.Users.UpdateUser(context.Background(), user); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	// Access /me with suspended user - should fail
	middleware := SessionAuthMiddleware(svc)
	wrapped := middleware(http.HandlerFunc(handler.Me))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+authResult.SessionToken)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	// The handler should return 401 because middleware doesn't inject user context for suspended users
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("suspended user should be rejected: status = %d, want 401", w.Code)
	}
}

func TestMiddleware_RejectsDeletedUser(t *testing.T) {
	handler, svc, pool := newTestAuthHandler(t)

	rawToken, err := svc.RequestMagicLink(context.Background(), "deleted-mw@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Delete the user
	pool.Users.DeleteUser(context.Background(), authResult.User.ID)

	middleware := SessionAuthMiddleware(svc)
	wrapped := middleware(http.HandlerFunc(handler.Me))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+authResult.SessionToken)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("deleted user should be rejected: status = %d, want 401", w.Code)
	}
}

// TestListSessions_RequiresAuth verifies that /auth/sessions requires authentication.
func TestListSessions_RequiresAuth(t *testing.T) {
	handler, _, _ := newTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
	w := httptest.NewRecorder()

	handler.ListSessions(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (unauthenticated should be rejected)", w.Code)
	}
}

// TestListSessions_WithAuthMiddleware verifies that /auth/sessions works with valid session.
func TestListSessions_WithAuthMiddleware(t *testing.T) {
	handler, svc, _ := newTestAuthHandler(t)

	// Login first
	rawToken, err := svc.RequestMagicLink(context.Background(), "sessions-list@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	middleware := SessionAuthMiddleware(svc)
	wrapped := middleware(http.HandlerFunc(handler.ListSessions))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+authResult.SessionToken)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if _, ok := resp["sessions"]; !ok {
		t.Fatal("expected sessions in response")
	}
}

// TestRevokeSession_RequiresAuth verifies that session revocation requires authentication.
func TestRevokeSession_RequiresAuth(t *testing.T) {
	handler, _, _ := newTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/some-session-id", nil)
	w := httptest.NewRecorder()

	handler.RevokeSession(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (unauthenticated should be rejected)", w.Code)
	}
}

// TestRevokeSession_WithAuthMiddleware verifies that session revocation works with valid auth.
func TestRevokeSession_WithAuthMiddleware(t *testing.T) {
	handler, svc, _ := newTestAuthHandler(t)

	// Login first
	rawToken, err := svc.RequestMagicLink(context.Background(), "sessions-revoke@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	middleware := SessionAuthMiddleware(svc)
	wrapped := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.RevokeSession(w, r)
	}))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/"+authResult.Session.ID, nil)
	req.Header.Set("Authorization", "Bearer "+authResult.SessionToken)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["message"] != "session revoked successfully" {
		t.Fatalf("unexpected message: %s", resp["message"])
	}
}

// TestRevokeSession_BadMethod returns 405 for non-DELETE.
func TestRevokeSession_BadMethod(t *testing.T) {
	handler, _, _ := newTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions/some-id", nil)
	w := httptest.NewRecorder()

	handler.RevokeSession(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

// TestRevokeSession_MissingID returns 400 for missing session ID.
func TestRevokeSession_MissingID(t *testing.T) {
	handler, svc, _ := newTestAuthHandler(t)

	rawToken, err := svc.RequestMagicLink(context.Background(), "revoke-missing-id@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	middleware := SessionAuthMiddleware(svc)
	wrapped := middleware(http.HandlerFunc(handler.RevokeSession))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/", nil)
	req.Header.Set("Authorization", "Bearer "+authResult.SessionToken)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (missing session ID)", w.Code)
	}
}
