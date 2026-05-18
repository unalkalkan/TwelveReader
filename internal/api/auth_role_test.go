package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/identity"
	_ "modernc.org/sqlite"
)

// TestRequireRole_AllowsAdmin tests that RequireRole("admin") passes for admin users.
func TestRequireRole_AllowsAdmin(t *testing.T) {
	_, svc, pool := newTestAuthHandlerWithPool(t)

	// Login as user, then promote to admin
	rawToken, err := svc.RequestMagicLink(context.Background(), "admin-role-test@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Promote to admin role
	adminRole, err := pool.Roles.GetRoleByName(context.Background(), "admin")
	if err != nil {
		t.Fatalf("GetRoleByName(admin): %v", err)
	}
	authResult.User.RoleID = adminRole.ID

	middleware := RequireRole(pool, "admin")
	wrapped := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), sessionIDKey{}, authResult.Session.ID)
	ctx = context.WithValue(ctx, userContextKey{}, authResult.User)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (admin should pass RequireRole(admin))", w.Code)
	}
}

// TestRequireRole_DeniesUser tests that RequireRole("admin") rejects regular users.
func TestRequireRole_DeniesUser(t *testing.T) {
	_, svc, pool := newTestAuthHandlerWithPool(t)

	// Login as regular user (default role is "user", not "admin")
	rawToken, err := svc.RequestMagicLink(context.Background(), "regular-role-test@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	middleware := RequireRole(pool, "admin")
	wrapped := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), sessionIDKey{}, authResult.Session.ID)
	ctx = context.WithValue(ctx, userContextKey{}, authResult.User)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (regular user should be denied admin role)", w.Code)
	}
}

// TestRequireRole_DeniesUnauthenticated tests that RequireRole rejects unauthenticated requests.
func TestRequireRole_DeniesUnauthenticated(t *testing.T) {
	_, _, pool := newTestAuthHandlerWithPool(t)

	middleware := RequireRole(pool, "admin")
	wrapped := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (unauthenticated should be rejected)", w.Code)
	}
}

// TestRequireAdminRole_AllowsAdmin tests the RequireAdminRole convenience wrapper.
func TestRequireAdminRole_AllowsAdmin(t *testing.T) {
	_, svc, pool := newTestAuthHandlerWithPool(t)

	rawToken, err := svc.RequestMagicLink(context.Background(), "admin-conc-test@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	adminRole, err := pool.Roles.GetRoleByName(context.Background(), "admin")
	if err != nil {
		t.Fatalf("GetRoleByName(admin): %v", err)
	}
	authResult.User.RoleID = adminRole.ID

	middleware := RequireAdminRole(pool)
	wrapped := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), sessionIDKey{}, authResult.Session.ID)
	ctx = context.WithValue(ctx, userContextKey{}, authResult.User)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// TestRequireAdminRole_DeniesNonAdmin tests RequireAdminRole blocks non-admin users.
func TestRequireAdminRole_DeniesNonAdmin(t *testing.T) {
	_, svc, pool := newTestAuthHandlerWithPool(t)

	rawToken, err := svc.RequestMagicLink(context.Background(), "nonadmin-test@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	middleware := RequireAdminRole(pool)
	wrapped := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), sessionIDKey{}, authResult.Session.ID)
	ctx = context.WithValue(ctx, userContextKey{}, authResult.User)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (non-admin should be denied)", w.Code)
	}
}

// TestMe_ResponseIncludesRoleName verifies that /auth/me includes role_name in response.
func TestMe_ResponseIncludesRoleName(t *testing.T) {
	handler, svc, pool := newTestAuthHandlerWithPool(t)

	// Login as admin
	rawToken, err := svc.RequestMagicLink(context.Background(), "me-role-admin@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	adminRole, err := pool.Roles.GetRoleByName(context.Background(), "admin")
	if err != nil {
		t.Fatalf("GetRoleByName(admin): %v", err)
	}
	authResult.User.RoleID = adminRole.ID

	middleware := SessionAuthMiddleware(svc)
	wrapped := middleware(http.HandlerFunc(handler.Me))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	ctx := context.WithValue(req.Context(), sessionIDKey{}, authResult.Session.ID)
	ctx = context.WithValue(ctx, userContextKey{}, authResult.User)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if roleName, ok := resp["role_name"].(string); !ok || roleName != "admin" {
		t.Fatalf("expected role_name=admin, got %v", resp["role_name"])
	}
}

// TestMe_ResponseShowsUserRole verifies that a regular user sees role_name=user.
func TestMe_ResponseShowsUserRole(t *testing.T) {
	handler, svc, _ := newTestAuthHandlerWithPool(t)

	rawToken, err := svc.RequestMagicLink(context.Background(), "me-role-user@example.com")
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
	ctx := context.WithValue(req.Context(), sessionIDKey{}, authResult.Session.ID)
	ctx = context.WithValue(ctx, userContextKey{}, authResult.User)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if roleName, ok := resp["role_name"].(string); !ok || roleName != "user" {
		t.Fatalf("expected role_name=user, got %v", resp["role_name"])
	}
}

// TestRequireAuth_ExpiredSession returns specific 401 for expired sessions.
func TestRequireAuth_ExpiredSession(t *testing.T) {
	_, svc, _ := newTestAuthHandlerWithPool(t)

	middleware := RequireAuth(svc)
	wrapped := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer expired-token-placeholder")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// newTestAuthHandlerWithPool creates a test AuthHandler with DBPool for role-based tests.
func newTestAuthHandlerWithPool(t *testing.T) (*AuthHandler, *identity.AuthService, *identity.DBPool) {
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
