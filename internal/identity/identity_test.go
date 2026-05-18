package identity

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
	_ "modernc.org/sqlite"
)

// newTestPool creates a fresh temp-file-backed SQLite DB pool for each test.
// modernc.org/sqlite :memory: databases are not shared across the connection
// pool, so we use a real file to ensure all connections see the same data.
func newTestPool(t *testing.T) *DBPool {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	pool, err := NewDBPool(dbPath)
	if err != nil {
		t.Fatalf("NewDBPool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func TestGenerateID(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateID()
		if id == "" {
			t.Fatal("GenerateID returned empty string")
		}
		if len(id) < 32 { // UUID v4 is at least 36 chars but just check > 32
			t.Fatalf("GenerateID too short: %s", id)
		}
		if ids[id] {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestHashToken(t *testing.T) {
	h1 := HashToken("same-token")
	h2 := HashToken("same-token")
	if h1 != h2 {
		t.Fatal("HashToken should be deterministic")
	}
	h3 := HashToken("different-token")
	if h1 == h3 {
		t.Fatal("HashToken should differ for different inputs")
	}
	// Verify it's not the original string (basic sanity)
	if h1 == "same-token" {
		t.Fatal("HashToken returned the plaintext")
	}
}

func TestAccountRepository(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	a := &types.Account{Name: "Test Account", Slug: "test-account"}
	err := pool.Accounts.CreateAccount(ctx, a)
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if a.ID == "" {
		t.Fatal("Account ID not set after create")
	}

	got, err := pool.Accounts.GetAccountByID(ctx, a.ID)
	if err != nil {
		t.Fatalf("GetAccountByID: %v", err)
	}
	if got.Name != "Test Account" {
		t.Fatalf("Name = %q, want %q", got.Name, "Test Account")
	}

	// Get by slug
	got2, err := pool.Accounts.GetAccountBySlug(ctx, "test-account")
	if err != nil {
		t.Fatalf("GetAccountBySlug: %v", err)
	}
	if got2.ID != a.ID {
		t.Fatal("GetAccountBySlug returned wrong account")
	}

	// List accounts
	list, err := pool.Accounts.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListAccounts count = %d, want 1", len(list))
	}

	// Update account
	got.Name = "Updated Account"
	err = pool.Accounts.UpdateAccount(ctx, got)
	if err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}

	// Delete account (soft delete)
	err = pool.Accounts.DeleteAccount(ctx, a.ID)
	if err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	// Should not appear in list anymore
	list2, err := pool.Accounts.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("ListAccounts after delete: %v", err)
	}
	if len(list2) != 0 {
		t.Fatalf("ListAccounts should be empty after soft delete, got %d", len(list2))
	}
}

func TestRoleRepository(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	r := &types.Role{Name: "admin", Description: "Administrator", IsSystem: true}
	err := pool.Roles.CreateRole(ctx, r)
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	// Get by name
	got, err := pool.Roles.GetRoleByName(ctx, "admin")
	if err != nil {
		t.Fatalf("GetRoleByName: %v", err)
	}
	if !got.IsSystem {
		t.Fatal("IsSystem should be true")
	}

	// Cannot delete system role
	err = pool.Roles.DeleteRole(ctx, r.ID)
	if err == nil {
		t.Fatal("Delete system role should fail")
	}

	// Create non-system role and delete it
	r2 := &types.Role{Name: "custom", Description: "Custom role", IsSystem: false}
	err = pool.Roles.CreateRole(ctx, r2)
	if err != nil {
		t.Fatalf("CreateRole custom: %v", err)
	}
	err = pool.Roles.DeleteRole(ctx, r2.ID)
	if err != nil {
		t.Fatalf("DeleteRole custom: %v", err)
	}

	// List should only have admin now
	list, err := pool.Roles.ListRoles(ctx)
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListRoles count = %d, want 1", len(list))
	}
}

func TestUserRepository(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	// Setup: create account and role first
	acct := &types.Account{Name: "Acme Corp", Slug: "acme"}
	if err := pool.Accounts.CreateAccount(ctx, acct); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	role := &types.Role{Name: "user", Description: "Regular user"}
	if err := pool.Roles.CreateRole(ctx, role); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	u := &types.User{AccountID: acct.ID, Email: "alice@example.com", RoleID: role.ID}
	err := pool.Users.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.ID == "" {
		t.Fatal("User ID not set after create")
	}

	// Get by email
	got, err := pool.Users.GetUserByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if got.Email != "alice@example.com" {
		t.Fatalf("Email = %q", got.Email)
	}

	// List by account
	list, err := pool.Users.ListUsersByAccount(ctx, acct.ID)
	if err != nil {
		t.Fatalf("ListUsersByAccount: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListUsersByAccount count = %d, want 1", len(list))
	}

	// Update user
	got.Name = "Alice Updated"
	err = pool.Users.UpdateUser(ctx, got)
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	// Delete (soft delete)
	err = pool.Users.DeleteUser(ctx, u.ID)
	if err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	// Should not be in list anymore
	list2, err := pool.Users.ListUsersByAccount(ctx, acct.ID)
	if err != nil {
		t.Fatalf("ListUsersByAccount after delete: %v", err)
	}
	if len(list2) != 0 {
		t.Fatalf("ListUsersByAccount should be empty, got %d", len(list2))
	}
}

func TestSessionRepository(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	// Setup user
	acct := &types.Account{Name: "Test", Slug: "test"}
	pool.Accounts.CreateAccount(ctx, acct)
	role := &types.Role{Name: "user", Description: "User"}
	pool.Roles.CreateRole(ctx, role)
	user := &types.User{AccountID: acct.ID, Email: "bob@example.com", RoleID: role.ID}
	pool.Users.CreateUser(ctx, user)

	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	s := &types.Session{UserID: user.ID, TokenHash: HashToken("session-token-1"), ExpiresAt: expiresAt}
	err := pool.Sessions.CreateSession(ctx, s)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Get by token hash
	got, err := pool.Sessions.GetSessionByTokenHash(ctx, HashToken("session-token-1"))
	if err != nil {
		t.Fatalf("GetSessionByTokenHash: %v", err)
	}
	if got.UserID != user.ID {
		t.Fatal("Session user_id mismatch")
	}

	// List active sessions
	active, err := pool.Sessions.ListActiveSessionsByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListActiveSessionsByUser: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("Active sessions count = %d, want 1", len(active))
	}

	// Revoke session
	err = pool.Sessions.RevokeSession(ctx, s.ID)
	if err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}

	// Should not be in active list anymore
	active2, err := pool.Sessions.ListActiveSessionsByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListActiveSessionsByUser after revoke: %v", err)
	}
	if len(active2) != 0 {
		t.Fatalf("Active sessions should be empty after revoke, got %d", len(active2))
	}

	// Create expired session and clean up
	expired := &types.Session{UserID: user.ID, TokenHash: HashToken("expired-session"), ExpiresAt: time.Now().UTC().Add(-1 * time.Hour)}
	pool.Sessions.CreateSession(ctx, expired)

	n, err := pool.Sessions.DeleteExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("DeleteExpiredSessions: %v", err)
	}
	if n < 2 { // revoked + expired
		t.Fatalf("DeleteExpiredSessions should delete at least 2, got %d", n)
	}
}

func TestRefreshTokenRepository(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	// Setup user
	acct := &types.Account{Name: "Test", Slug: "test-rt"}
	pool.Accounts.CreateAccount(ctx, acct)
	role := &types.Role{Name: "user", Description: "User"}
	pool.Roles.CreateRole(ctx, role)
	user := &types.User{AccountID: acct.ID, Email: "charlie@example.com", RoleID: role.ID}
	pool.Users.CreateUser(ctx, user)

	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	rt := &types.RefreshToken{UserID: user.ID, TokenHash: HashToken("refresh-1"), ExpiresAt: expiresAt}
	err := pool.RefreshTokens.CreateRefreshToken(ctx, rt)
	if err != nil {
		t.Fatalf("CreateRefreshToken: %v", err)
	}

	// Get by hash
	got, err := pool.RefreshTokens.GetRefreshTokenByHash(ctx, HashToken("refresh-1"))
	if err != nil {
		t.Fatalf("GetRefreshTokenByHash: %v", err)
	}
	if got.UserID != user.ID {
		t.Fatal("RefreshToken user_id mismatch")
	}

	// Revoke
	err = pool.RefreshTokens.RevokeRefreshToken(ctx, rt.ID)
	if err != nil {
		t.Fatalf("RevokeRefreshToken: %v", err)
	}

	// Should be revoked
	got2, _ := pool.RefreshTokens.GetRefreshTokenByID(ctx, rt.ID)
	if !got2.Revoked {
		t.Fatal("RefreshToken should be revoked")
	}
}

func TestAuditLogRepository(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	entry := &types.AuditLogEntry{
		UserID:      "user-123",
		AccountID:   "acct-456",
		EventType:   types.AuditEventLoginSuccess,
		Description: "User logged in via magic link",
		IPAddress:   "192.168.1.1",
		UserAgent:   "Mozilla/5.0",
		Metadata:    map[string]string{"method": "magic_link"},
	}
	err := pool.AuditLog.CreateEntry(ctx, entry)
	if err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if entry.ID == "" {
		t.Fatal("Audit entry ID not set")
	}

	// Get by ID
	got, err := pool.AuditLog.GetEntryByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetEntryByID: %v", err)
	}
	if got.EventType != types.AuditEventLoginSuccess {
		t.Fatalf("EventType = %q", got.EventType)
	}

	// List by user
	list1, err := pool.AuditLog.ListEntriesByUser(ctx, "user-123", 10)
	if err != nil {
		t.Fatalf("ListEntriesByUser: %v", err)
	}
	if len(list1) != 1 {
		t.Fatalf("ListEntriesByUser count = %d, want 1", len(list1))
	}

	// List by type
	list2, err := pool.AuditLog.ListEntriesByType(ctx, types.AuditEventLoginSuccess, 10)
	if err != nil {
		t.Fatalf("ListEntriesByType: %v", err)
	}
	if len(list2) != 1 {
		t.Fatalf("ListEntriesByType count = %d, want 1", len(list2))
	}

	// List recent
	list3, err := pool.AuditLog.ListRecentEntries(ctx, 10)
	if err != nil {
		t.Fatalf("ListRecentEntries: %v", err)
	}
	if len(list3) != 1 {
		t.Fatalf("ListRecentEntries count = %d, want 1", len(list3))
	}

	// Metadata round-trip
	if got.Metadata == nil || got.Metadata["method"] != "magic_link" {
		t.Fatalf("Metadata not preserved: %+v", got.Metadata)
	}

	// Test with empty metadata
	entry2 := &types.AuditLogEntry{
		UserID:    "user-456",
		EventType: types.AuditEventLogout,
	}
	err = pool.AuditLog.CreateEntry(ctx, entry2)
	if err != nil {
		t.Fatalf("CreateEntry with no metadata: %v", err)
	}

	// Test null user_id / account_id handling
	entry3 := &types.AuditLogEntry{
		UserID:      "",
		EventType:   types.AuditEventAdminAction,
		Description: "System bootstrap",
	}
	err = pool.AuditLog.CreateEntry(ctx, entry3)
	if err != nil {
		t.Fatalf("CreateEntry with empty user/account: %v", err)
	}

	// Verify recent entries count
	list4, err := pool.AuditLog.ListRecentEntries(ctx, 10)
	if err != nil {
		t.Fatalf("ListRecentEntries: %v", err)
	}
	if len(list4) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(list4))
	}
}

func TestMigrationIdempotency(t *testing.T) {
	pool := newTestPool(t)
	// Run migrations again - should be idempotent
	err := pool.runMigrations()
	if err != nil {
		t.Fatalf("Re-running migrations failed: %v", err)
	}

	// Verify tables still work
	ctx := context.Background()
	a := &types.Account{Name: "Idem Test", Slug: "idem-test"}
	err = pool.Accounts.CreateAccount(ctx, a)
	if err != nil {
		t.Fatalf("CreateAccount after re-migration: %v", err)
	}
}

func TestDBPoolClose(t *testing.T) {
	pool := newTestPool(t)
	err := pool.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
}
