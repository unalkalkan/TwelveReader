package identity

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestSessionLifecycle_RefreshRevokesPrevious verifies that RefreshSession revokes the previous
// session paired with the consumed refresh token.
func TestSessionLifecycle_RefreshRevokesPrevious(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	// Login
	rawToken, err := svc.RequestMagicLink(ctx, "lifecycle@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	oldSessionID := authResult.Session.ID
	oldSessionToken := authResult.SessionToken

	// Verify old session is valid
	_, err = svc.GetSessionByTokenHash(ctx, oldSessionToken)
	if err != nil {
		t.Fatalf("old session should be valid before refresh: %v", err)
	}

	// Refresh
	newResult, err := svc.RefreshSession(ctx, authResult.RefreshToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("RefreshSession: %v", err)
	}

	// Old session should be revoked
	_, err = svc.GetSessionByTokenHash(ctx, oldSessionToken)
	if err == nil {
		t.Fatal("old session should be revoked after refresh")
	}
	if !IsSessionRevoked(err) {
		t.Fatalf("expected revoked error, got: %v", err)
	}

	// New session should be valid
	newSession, err := svc.GetSessionByTokenHash(ctx, newResult.SessionToken)
	if err != nil {
		t.Fatalf("new session should be valid: %v", err)
	}
	if newSession.UserID != authResult.User.ID {
		t.Fatal("new session user mismatch")
	}

	// Old session ID should still exist but revoked
	oldSession, err := svc.pool.Sessions.GetSessionByID(ctx, oldSessionID)
	if err != nil {
		t.Fatalf("old session should still be queryable: %v", err)
	}
	if !oldSession.Revoked {
		t.Fatal("old session should be revoked in DB")
	}

	// Refresh with same token again should fail (one-time use)
	_, err = svc.RefreshSession(ctx, authResult.RefreshToken, "127.0.0.1", "TestClient/1.0")
	if err == nil {
		t.Fatal("second refresh with same token should fail")
	}
}

// TestSessionLifecycle_ChainedRefresh verifies multiple consecutive refreshes work correctly.
func TestSessionLifecycle_ChainedRefresh(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	// Login
	rawToken, err := svc.RequestMagicLink(ctx, "chain@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Chain 3 refreshes
	var currentResult *AuthResult = authResult
	for i := 0; i < 3; i++ {
		newResult, err := svc.RefreshSession(ctx, currentResult.RefreshToken, "127.0.0.1", "TestClient/1.0")
		if err != nil {
			t.Fatalf("RefreshSession %d: %v", i+1, err)
		}

		// Previous session should be revoked
		_, err = svc.GetSessionByTokenHash(ctx, currentResult.SessionToken)
		if err == nil {
			t.Fatalf("refresh %d: previous session should be revoked", i+1)
		}

		currentResult = newResult
	}

	// Final session should be valid
	session, err := svc.GetSessionByTokenHash(ctx, currentResult.SessionToken)
	if err != nil {
		t.Fatalf("final session should be valid: %v", err)
	}
	if session.UserID == "" {
		t.Fatal("final session has no user")
	}
}

// TestCleanupExpiredSessionsAndTokens verifies the cleanup operation removes stale data.
func TestCleanupExpiredSessionsAndTokens(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	// Login to create sessions and tokens
	rawToken, err := svc.RequestMagicLink(ctx, "cleanup@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Expire the session and refresh token manually
	_, err = svc.pool.DB().ExecContext(ctx, `
		UPDATE sessions SET expires_at = '2000-01-01T00:00:00Z' WHERE id = ?`, authResult.Session.ID)
	if err != nil {
		t.Fatalf("expire session: %v", err)
	}
	_, err = svc.pool.DB().ExecContext(ctx, `
		UPDATE refresh_tokens SET expires_at = '2000-01-01T00:00:00Z' WHERE id = ?`, authResult.RefreshRT.ID)
	if err != nil {
		t.Fatalf("expire refresh token: %v", err)
	}

	// Run cleanup
	result, err := svc.CleanupExpiredSessionsAndTokens(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredSessionsAndTokens: %v", err)
	}

	if result.SessionsDeleted < 1 {
		t.Fatalf("expected at least 1 session deleted, got %d", result.SessionsDeleted)
	}
	if result.RefreshTokensDeleted < 1 {
		t.Fatalf("expected at least 1 refresh token deleted, got %d", result.RefreshTokensDeleted)
	}

	// Verify data is actually gone
	_, err = svc.pool.Sessions.GetSessionByID(ctx, authResult.Session.ID)
	if err == nil {
		t.Fatal("expired session should be deleted")
	}
}

// TestListUserSessions verifies that active sessions are listed correctly.
func TestListUserSessions(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	// Login
	rawToken, err := svc.RequestMagicLink(ctx, "list@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// List sessions
	sessions, err := svc.ListUserSessions(ctx, authResult.User.ID)
	if err != nil {
		t.Fatalf("ListUserSessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ID != authResult.Session.ID {
		t.Fatal("session ID mismatch")
	}

	// Logout and verify list is empty
	err = svc.Logout(ctx, authResult.Session.ID)
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}

	sessions, err = svc.ListUserSessions(ctx, authResult.User.ID)
	if err != nil {
		t.Fatalf("ListUserSessions after logout: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions after logout, got %d", len(sessions))
	}
}

// TestRevokeSpecificSession verifies that a user can revoke their own specific session.
func TestRevokeSpecificSession(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	// Login twice (two sessions for same user)
	rawToken1, err := svc.RequestMagicLink(ctx, "revoke@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	result1, err := svc.VerifyMagicLink(ctx, rawToken1, "127.0.0.1", "Device A")
	if err != nil {
		t.Fatalf("VerifyMagicLink 1: %v", err)
	}

	rawToken2, err := svc.RequestMagicLink(ctx, "revoke@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink 2: %v", err)
	}
	result2, err := svc.VerifyMagicLink(ctx, rawToken2, "192.168.1.1", "Device B")
	if err != nil {
		t.Fatalf("VerifyMagicLink 2: %v", err)
	}

	// Both sessions should be active
	sessions, _ := svc.ListUserSessions(ctx, result1.User.ID)
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Revoke session 1 only
	err = svc.RevokeSpecificSession(ctx, result1.Session.ID, result1.User.ID)
	if err != nil {
		t.Fatalf("RevokeSpecificSession: %v", err)
	}

	// Session 1 should be revoked
	_, err = svc.GetSessionByTokenHash(ctx, result1.SessionToken)
	if err == nil {
		t.Fatal("session 1 should be revoked")
	}

	// Session 2 should still be valid
	session2, err := svc.GetSessionByTokenHash(ctx, result2.SessionToken)
	if err != nil {
		t.Fatalf("session 2 should still be valid: %v", err)
	}
	if session2.ID != result2.Session.ID {
		t.Fatal("session 2 ID mismatch")
	}

	// Active sessions should be 1
	sessions, _ = svc.ListUserSessions(ctx, result1.User.ID)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 active session, got %d", len(sessions))
	}

	// Cross-user revocation should fail
	err = svc.RevokeSpecificSession(ctx, result2.Session.ID, "nonexistent-user-id")
	if err == nil {
		t.Fatal("cross-user revocation should fail")
	}
	if !strings.Contains(err.Error(), "does not belong to this user") {
		t.Fatalf("unexpected error: %v", err)
	}

	// Revoke already-revoked session should succeed (no-op)
	err = svc.RevokeSpecificSession(ctx, result1.Session.ID, result1.User.ID)
	if err != nil {
		t.Fatalf("revoke already-revoked session should be no-op: %v", err)
	}
}

// TestSessionErrorTypes verifies that the error types are correctly exposed.
func TestSessionErrorTypes(t *testing.T) {
	// ErrSessionExpired
	if errors.Is(ErrSessionExpired, ErrSessionExpired) != true {
		t.Fatal("ErrSessionExpired should match itself")
	}

	// ErrSessionRevoked
	if errors.Is(ErrSessionRevoked, ErrSessionRevoked) != true {
		t.Fatal("ErrSessionRevoked should match itself")
	}

	// SessionError struct
	se := newSessionError("expired", "session expired")
	if se.Cause != "expired" || se.Message != "session expired" {
		t.Fatalf("SessionError fields wrong: %+v", se)
	}
	if se.Error() != "session expired" {
		t.Fatalf("SessionError.Error() = %q", se.Error())
	}

	// IsSessionExpired
	if !IsSessionExpired(ErrSessionExpired) {
		t.Fatal("ErrSessionExpired should be detected as expired")
	}
	if IsSessionExpired(ErrSessionRevoked) {
		t.Fatal("ErrSessionRevoked should NOT be detected as expired")
	}

	// IsSessionRevoked
	if !IsSessionRevoked(ErrSessionRevoked) {
		t.Fatal("ErrSessionRevoked should be detected as revoked")
	}
	if IsSessionRevoked(ErrSessionExpired) {
		t.Fatal("ErrSessionExpired should NOT be detected as revoked")
	}
}

// TestGetSessionByTokenHash_InvalidToken verifies that an invalid (non-existent) token
// returns a proper error, not Expired or Revoked.
func TestGetSessionByTokenHash_InvalidToken(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	_, err := svc.GetSessionByTokenHash(ctx, "totally-fake-token-that-does-not-exist")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if errors.Is(err, ErrSessionExpired) {
		t.Fatal("invalid token should not be ErrSessionExpired")
	}
	if errors.Is(err, ErrSessionRevoked) {
		t.Fatal("invalid token should not be ErrSessionRevoked")
	}
}

// TestGetSessionByTokenHash_RevokedSession verifies that a revoked session returns ErrSessionRevoked.
func TestGetSessionByTokenHash_RevokedSession(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	rawToken, err := svc.RequestMagicLink(ctx, "revoked-test@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Revoke the session
	err = svc.pool.Sessions.RevokeSession(ctx, authResult.Session.ID)
	if err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}

	_, err = svc.GetSessionByTokenHash(ctx, authResult.SessionToken)
	if err == nil {
		t.Fatal("expected error for revoked session")
	}
	if !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("expected ErrSessionRevoked, got: %v", err)
	}
	if !IsSessionRevoked(err) {
		t.Fatal("IsSessionRevoked should detect this error")
	}
}

// TestGetSessionByTokenHash_ExpiredSessionViaUpdate verifies that an expired session returns ErrSessionExpired.
func TestGetSessionByTokenHash_ExpiredSessionViaUpdate(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	rawToken, err := svc.RequestMagicLink(ctx, "expired-test@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Expire the session manually
	_, err = svc.pool.DB().ExecContext(ctx, `
		UPDATE sessions SET expires_at = '2000-01-01T00:00:00Z' WHERE id = ?`, authResult.Session.ID)
	if err != nil {
		t.Fatalf("expire session: %v", err)
	}

	_, err = svc.GetSessionByTokenHash(ctx, authResult.SessionToken)
	if err == nil {
		t.Fatal("expected error for expired session")
	}
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got: %v", err)
	}
	if !IsSessionExpired(err) {
		t.Fatal("IsSessionExpired should detect this error")
	}
}

// TestRefreshSession_PreviousSessionRevokedOnRefresh verifies the audit log entry.
func TestRefreshSession_AuditLog(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	rawToken, err := svc.RequestMagicLink(ctx, "audit@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Refresh
	_, err = svc.RefreshSession(ctx, authResult.RefreshToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("RefreshSession: %v", err)
	}

	// Check audit log has the refresh event
	entries, err := svc.pool.AuditLog.ListEntriesByUser(ctx, authResult.User.ID, 50)
	if err != nil {
		t.Fatalf("ListEntriesByUser: %v", err)
	}

	foundRefresh := false
	foundRevoke := false
	for _, entry := range entries {
		if strings.Contains(entry.Description, "session_refreshed") {
			foundRefresh = true
		}
		if strings.Contains(entry.Description, "previous_session_revoked_on_refresh") {
			foundRevoke = true
		}
	}

	if !foundRefresh {
		t.Fatal("audit log missing session_refreshed entry")
	}
	if !foundRevoke {
		t.Fatal("audit log missing previous_session_revoked_on_refresh entry")
	}
}

// TestSessionLifecycle_LogoutThenRefreshTokenFails verifies that after logout,
// the paired refresh token is also revoked.
func TestSessionLifecycle_LogoutThenRefreshTokenFails(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	rawToken, err := svc.RequestMagicLink(ctx, "logout-rt@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Logout
	err = svc.Logout(ctx, authResult.Session.ID)
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}

	// Refresh token should fail (consumed + revoked by logout)
	_, err = svc.RefreshSession(ctx, authResult.RefreshToken, "127.0.0.1", "TestClient/1.0")
	if err == nil {
		t.Fatal("refresh with logged-out token should fail")
	}
}

// TestRevokeAllUserSessions verifies that revoking all user sessions also revokes paired refresh tokens.
func TestRevokeAllUserSessions_RevokesRefreshTokens(t *testing.T) {
	svc, _ := newTestAuthService(t)
	ctx := context.Background()

	rawToken, err := svc.RequestMagicLink(ctx, "revoke-all@example.com")
	if err != nil {
		t.Fatalf("RequestMagicLink: %v", err)
	}
	authResult, err := svc.VerifyMagicLink(ctx, rawToken, "127.0.0.1", "TestClient/1.0")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}

	// Revoke all sessions
	err = svc.RevokeAllUserSessions(ctx, authResult.User.ID)
	if err != nil {
		t.Fatalf("RevokeAllUserSessions: %v", err)
	}

	// Session should be revoked
	_, err = svc.GetSessionByTokenHash(ctx, authResult.SessionToken)
	if err == nil {
		t.Fatal("session should be revoked")
	}

	// Refresh token should also be revoked
	rt, err := svc.pool.RefreshTokens.GetRefreshTokenByID(ctx, authResult.RefreshRT.ID)
	if err != nil {
		t.Fatalf("GetRefreshTokenByID: %v", err)
	}
	if !rt.Revoked {
		t.Fatal("refresh token should be revoked by RevokeAllUserSessions")
	}

	// Active sessions list should be empty
	sessions, _ := svc.ListUserSessions(ctx, authResult.User.ID)
	if len(sessions) != 0 {
		t.Fatalf("expected 0 active sessions, got %d", len(sessions))
	}
}
