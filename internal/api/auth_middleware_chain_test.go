package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

// TestWrapAdminChain_RejectsWhenOrderWrong demonstrates that the middleware
// composition order used in main.go is broken:
//   RequireAdminRole(pool)(RequireAuth(svc)(handler))
// RequireAdminRole runs first and checks the user from context, but RequireAuth
// hasn't run yet — so the user is nil and every request gets 401.
func TestWrapAdminChain_RejectsWhenOrderWrong(t *testing.T) {
	_, svc, pool := newTestAuthHandlerWithPool(t)

	// Login as admin
	rawToken, err := svc.RequestMagicLink(context.Background(), "admin-chain-wrong@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Promote to admin
	adminRole, err := pool.Roles.GetRoleByName(context.Background(), "admin")
	if err != nil {
		t.Fatalf("GetRoleByName(admin): %v", err)
	}
	authResult.User.RoleID = adminRole.ID

	// The BROKEN order used in main.go:
	//   RequireAdminRole(pool)(RequireAuth(svc)(handler))
	// When the request arrives:
	//   1. RequireAdminRole checks user in context — nil, returns 401
	//   2. RequireAuth NEVER runs — user never gets injected
	brokenChain := RequireAdminRole(pool)(RequireAuth(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+authResult.SessionToken)
	w := httptest.NewRecorder()

	brokenChain.ServeHTTP(w, req)

	// Regression: the broken order always returns 401 because RequireAdminRole
	// checks user from context before RequireAuth has injected it.
	// This test verifies that wrong order = 401 for everyone (even admins).
	if w.Code != http.StatusUnauthorized {
		t.Errorf("BROKEN CHAIN: admin with valid token got %d, want 401. "+
			"If this changes the broken chain is no longer broken — update this test.", w.Code)
	}
}

// TestWrapAdminChain_PassesWhenOrderCorrect demonstrates the fixed order:
//   RequireAuth(svc)(RequireAdminRole(pool)(handler))
// RequireAuth runs first, sets user in context, then RequireAdminRole checks it.
func TestWrapAdminChain_PassesWhenOrderCorrect(t *testing.T) {
	_, svc, pool := newTestAuthHandlerWithPool(t)

	// Login as admin
	rawToken, err := svc.RequestMagicLink(context.Background(), "admin-chain-correct@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Promote to admin in DB (RequireAuth fetches fresh user from DB on each request)
	adminRole, err := pool.Roles.GetRoleByName(context.Background(), "admin")
	if err != nil {
		t.Fatalf("GetRoleByName(admin): %v", err)
	}
	_, err = pool.DB().ExecContext(context.Background(),
		"UPDATE users SET role_id = ? WHERE id = ?", adminRole.ID, authResult.User.ID)
	if err != nil {
		t.Fatalf("promote to admin in DB: %v", err)
	}

	// CORRECT order: RequireAuth first, then RequireAdminRole
	correctChain := RequireAuth(svc)(RequireAdminRole(pool)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+authResult.SessionToken)
	w := httptest.NewRecorder()

	correctChain.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CORRECT CHAIN: admin with valid token got %d, want 200. "+
			"body: %s", w.Code, w.Body.String())
	}
}

// TestWrapAdminChain_CorrectOrderDeniesNonAdmin verifies the correct chain
// still rejects non-admin users.
func TestWrapAdminChain_CorrectOrderDeniesNonAdmin(t *testing.T) {
	_, svc, pool := newTestAuthHandlerWithPool(t)

	// Login as regular user
	rawToken, err := svc.RequestMagicLink(context.Background(), "admin-chain-nonadmin@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(context.Background(), rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Correct order
	correctChain := RequireAuth(svc)(RequireAdminRole(pool)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+authResult.SessionToken)
	w := httptest.NewRecorder()

	correctChain.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("CORRECT CHAIN: non-admin user got %d, want 403. body: %s",
			w.Code, w.Body.String())
	}
}

// TestWrapAdminChain_CorrectOrderDeniesNoToken verifies the correct chain
// rejects requests without a token.
func TestWrapAdminChain_CorrectOrderDeniesNoToken(t *testing.T) {
	_, svc, pool := newTestAuthHandlerWithPool(t)

	correctChain := RequireAuth(svc)(RequireAdminRole(pool)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	w := httptest.NewRecorder()

	correctChain.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("CORRECT CHAIN: no token got %d, want 401. body: %s",
			w.Code, w.Body.String())
	}
}
