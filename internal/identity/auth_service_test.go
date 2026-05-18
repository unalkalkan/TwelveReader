package identity

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

func newTestAuthService(t *testing.T) (*AuthService, *DBPool) {
	t.Helper()
	pool := newTestPool(t)
	sender := &LogEmailSender{}
	return NewAuthService(pool, sender, "http://localhost:3000", "noreply@example.com",
		24*time.Hour, 7*24*time.Hour, 15*time.Minute), pool
}

func TestRequestMagicLink_ValidEmail(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	token, err := svc.RequestMagicLink(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	if len(token) != 64 { // 32 bytes hex
		t.Fatalf("token length = %d, want 64", len(token))
	}

	// Verify user was auto-created
	user, err := svc.pool.Users.GetUserByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if user.Status != "active" {
		t.Fatalf("user status = %q, want active", user.Status)
	}

	// Verify magic link was stored
	link, err := svc.pool.MagicLinks.GetMagicLinkByTokenHash(ctx, HashToken(token))
	if err != nil {
		t.Fatalf("GetMagicLinkByTokenHash: %v", err)
	}
	if link.Used {
		t.Fatal("link should not be used yet")
	}
	if !time.Now().UTC().Before(link.ExpiresAt) {
		t.Fatal("link already expired?")
	}
}

func TestRequestMagicLink_InvalidEmail(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	_, err := svc.RequestMagicLink(ctx, "not-an-email")
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
	if !strings.Contains(err.Error(), "invalid email") {
		t.Fatalf("error = %v", err)
	}
}

func TestRequestMagicLink_IdempotentUserCreation(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	// First request creates user
	token1, err := svc.RequestMagicLink(ctx, "bob@example.com")
	if err != nil {
		t.Fatalf("first RequestMagicLink: %v", err)
	}
	userID1, _ := getUserIdByEmail(ctx, svc.pool, "bob@example.com")

	// Second request should not create a new user
	token2, err := svc.RequestMagicLink(ctx, "bob@example.com")
	if err != nil {
		t.Fatalf("second RequestMagicLink: %v", err)
	}
	userID2, _ := getUserIdByEmail(ctx, svc.pool, "bob@example.com")

	if userID1 != userID2 {
		t.Fatal("user ID changed on second request — should be same user")
	}
	if token1 == token2 {
		t.Fatal("tokens should differ for separate requests")
	}
}

func TestVerifyMagicLink_HappyPath(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	// Request magic link
	rawToken, err := svc.RequestMagicLink(ctx, "carol@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}

	// Verify the link
	result, err := svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	if result.User == nil {
		t.Fatal("user is nil")
	}
	if result.User.Email != "carol@example.com" {
		t.Fatalf("email = %q", result.User.Email)
	}
	if result.SessionToken == "" {
		t.Fatal("session token empty")
	}
	if result.RefreshToken == "" {
		t.Fatal("refresh token empty")
	}

	// Verify magic link is now marked as used
	link, err := svc.pool.MagicLinks.GetMagicLinkByTokenHash(ctx, HashToken(rawToken))
	if err != nil {
		t.Fatalf("GetMagicLinkByTokenHash: %v", err)
	}
	if !link.Used {
		t.Fatal("link should be marked used")
	}
}

func TestVerifyMagicLink_AlreadyUsed(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	rawToken, err := svc.RequestMagicLink(ctx, "dave@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}

	// First verify succeeds
	_, err = svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("first VerifyMagicLink: %v", err)
	}

	// Second verify should fail
	_, err = svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err == nil {
		t.Fatal("expected error for already-used link")
	}
	if !strings.Contains(err.Error(), "already used") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyMagicLink_Expired(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	// Set expiry to 1 second
	svc.linkExpiry = 1 * time.Second

	rawToken, err := svc.RequestMagicLink(ctx, "eve@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}

	time.Sleep(2 * time.Second)

	_, err = svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err == nil {
		t.Fatal("expected error for expired link")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyMagicLink_InvalidToken(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	_, err := svc.VerifyMagicLink(ctx, "totally-invalid-token-here", "127.0.0.1", "TestClient/1.0")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestRefreshSession_HappyPath(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	// Login first
	rawToken, err := svc.RequestMagicLink(ctx, "frank@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Refresh
	newResult, err := svc.RefreshSession(ctx, authResult.RefreshToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("RefreshSession: %v", err)
	}

	if newResult.SessionToken == authResult.SessionToken {
		t.Fatal("session token should change on refresh")
	}
	if newResult.RefreshToken == authResult.RefreshToken {
		t.Fatal("refresh token should change on refresh (rotation)")
	}

	// Old refresh token should be consumed
	_, err = svc.RefreshSession(ctx, authResult.RefreshToken, "127.0.0.1", "TestClient/1.0")
	if err == nil {
		t.Fatal("expected error for already-consumed refresh token")
	}
}

func TestRefreshSession_Expired(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	rawToken, err := svc.RequestMagicLink(ctx, "grace@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Expire the refresh token manually
	_, err = svc.pool.DB().ExecContext(ctx, `
		UPDATE refresh_tokens SET expires_at = '2000-01-01T00:00:00Z' 
		WHERE user_id = ?`, authResult.User.ID)
	if err != nil {
		t.Fatalf("expire refresh token: %v", err)
	}

	_, err = svc.RefreshSession(ctx, authResult.RefreshToken, "127.0.0.1", "TestClient/1.0")
	if err == nil {
		t.Fatal("expected error for expired refresh token")
	}
}

func TestLogout_RevokesSession(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	rawToken, err := svc.RequestMagicLink(ctx, "hank@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	sessionID := authResult.Session.ID

	// Logout
	err = svc.Logout(ctx, sessionID)
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}

	// Session should be revoked
	_, err = svc.GetSessionByTokenHash(ctx, authResult.SessionToken)
	if err == nil {
		t.Fatal("expected error after logout")
	}

	// Refresh tokens should also be revoked
	rt, err := svc.pool.RefreshTokens.GetRefreshTokenByHash(ctx, HashToken(authResult.RefreshToken))
	if err != nil {
		t.Fatalf("GetRefreshTokenByHash: %v", err)
	}
	if !rt.Revoked {
		t.Fatal("refresh token should be revoked after logout")
	}
}

func TestGetSessionByTokenHash_ExpiredSession(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	rawToken, err := svc.RequestMagicLink(ctx, "ivy@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Expire session manually
	_, err = svc.pool.DB().ExecContext(ctx, `
		UPDATE sessions SET expires_at = '2000-01-01T00:00:00Z' 
		WHERE id = ?`, authResult.Session.ID)
	if err != nil {
		t.Fatalf("expire session: %v", err)
	}

	_, err = svc.GetSessionByTokenHash(ctx, authResult.SessionToken)
	if err == nil {
		t.Fatal("expected error for expired session")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("error = %v", err)
	}
}

func TestAuthService_FullFlow(t *testing.T) {
	/*
		Request -> Verify -> Me (session valid) -> Refresh -> Logout -> Session invalid
	*/
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	// 1. Request magic link
	rawToken, err := svc.RequestMagicLink(ctx, "fullflow@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}

	// 2. Verify -> get session + refresh tokens
	authResult, err := svc.VerifyMagicLink(ctx, rawToken, "10.0.0.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	sessionToken1 := authResult.SessionToken
	refreshToken1 := authResult.RefreshToken
	sessionID := authResult.Session.ID

	// 3. Session is valid
	session, err := svc.GetSessionByTokenHash(ctx, sessionToken1)
	if err != nil {
		t.Fatalf("GetSessionByTokenHash: %v", err)
	}
	if session.UserID != authResult.User.ID {
		t.Fatal("session user mismatch")
	}

	// 4. Refresh -> new tokens
	newAuth, err := svc.RefreshSession(ctx, refreshToken1, "10.0.0.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("RefreshSession: %v", err)
	}
	sessionToken2 := newAuth.SessionToken
	if sessionToken1 == sessionToken2 {
		t.Fatal("session token should rotate on refresh")
	}

	// 5. Old session token is still valid (we didn't revoke it, just issued a new one)
	_, err = svc.GetSessionByTokenHash(ctx, sessionToken1)
	if err != nil {
		t.Fatalf("old session should still be valid: %v", err)
	}

	// 6. Logout revokes everything
	err = svc.Logout(ctx, sessionID)
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}

	// 7. Session no longer valid
	_, err = svc.GetSessionByTokenHash(ctx, sessionToken1)
	if err == nil {
		t.Fatal("session should be invalid after logout")
	}
}

func TestEnsureBootstrapAccount_Idempotent(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	// First call creates it
	acct1, err := svc.ensureBootstrapAccount(ctx)
	if err != nil {
		t.Fatalf("first ensureBootstrapAccount: %v", err)
	}

	// Second call returns the same account
	acct2, err := svc.ensureBootstrapAccount(ctx)
	if err != nil {
		t.Fatalf("second ensureBootstrapAccount: %v", err)
	}

	if acct1.ID != acct2.ID {
		t.Fatal("bootstrap account ID should be stable")
	}
}

func TestEnsureSystemRoles(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	svc.ensureBootstrapAccount(ctx) // triggers role creation

	admin, err := svc.pool.Roles.GetRoleByName(ctx, "admin")
	if err != nil {
		t.Fatalf("GetRoleByName admin: %v", err)
	}
	if !admin.IsSystem {
		t.Fatal("admin should be a system role")
	}

	user, err := svc.pool.Roles.GetRoleByName(ctx, "user")
	if err != nil {
		t.Fatalf("GetRoleByName user: %v", err)
	}
	if !user.IsSystem {
		t.Fatal("user should be a system role")
	}
}

func TestMagicLinkRepository(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	ml := &types.MagicLink{
		ID:        GenerateID(),
		Email:     "test@example.com",
		TokenHash: HashToken("some-token"),
		ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
		CreatedAt: time.Now().UTC(),
	}

	err := pool.MagicLinks.CreateMagicLink(ctx, ml)
	if err != nil {
		t.Fatalf("CreateMagicLink: %v", err)
	}

	got, err := pool.MagicLinks.GetMagicLinkByTokenHash(ctx, HashToken("some-token"))
	if err != nil {
		t.Fatalf("GetMagicLinkByTokenHash: %v", err)
	}
	if got.Email != "test@example.com" {
		t.Fatalf("email = %q", got.Email)
	}
	if got.Used {
		t.Fatal("should not be used yet")
	}

	// Mark used
	err = pool.MagicLinks.MarkUsed(ctx, ml.ID)
	if err != nil {
		t.Fatalf("MarkUsed: %v", err)
	}

	got2, err := pool.MagicLinks.GetMagicLinkByTokenHash(ctx, HashToken("some-token"))
	if err != nil {
		t.Fatalf("GetMagicLinkByTokenHash after mark: %v", err)
	}
	if !got2.Used {
		t.Fatal("should be used now")
	}

	// Delete expired/used links
	n, err := pool.MagicLinks.DeleteExpiredLinks(ctx)
	if err != nil {
		t.Fatalf("DeleteExpiredLinks: %v", err)
	}
	if n < 1 {
		t.Fatalf("DeleteExpiredLinks should delete at least 1, got %d", n)
	}
}

// Helper to get user ID by email for tests.
func getUserIdByEmail(ctx context.Context, pool *DBPool, email string) (string, error) {
	user, err := pool.Users.GetUserByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}
