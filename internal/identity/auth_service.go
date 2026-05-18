package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/mail"
	"strings"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// EmailSender is an interface for sending magic link emails.
type EmailSender interface {
	SendMagicLink(to, subject, body string) error
}

// LogEmailSender logs emails instead of actually sending them (for dev/local).
// NOTE: The body is redacted in production; set DevMode=true only in development.
type LogEmailSender struct {
	DevMode bool      // If true, log full email body (dev only). Default: false.
	output  io.Writer // For testing: captures log output instead of/stderr via log.Printf.
}

func (l *LogEmailSender) SendMagicLink(to, subject, body string) error {
	if l.DevMode {
		log.Printf("[EMAIL][DEV] To: %s, Subject: %s, Body:\n%s", to, subject, body)
		return nil
	}
	// Production-safe: log metadata only, never the token-bearing body.
	msg := fmt.Sprintf("[EMAIL] To: %s, Subject: %s (body suppressed in non-dev mode)", to, subject)
	if l.output != nil {
		fmt.Fprint(l.output, msg)
	} else {
		log.Print(msg)
	}
	return nil
}

// AuthService handles authentication business logic.
type AuthService struct {
	pool        *DBPool
	emailSender EmailSender
	baseURL     string
	senderFrom  string
	sessionTTL  time.Duration
	refreshTTL  time.Duration
	linkExpiry  time.Duration
}

// NewAuthService creates a new AuthService with the given config.
func NewAuthService(pool *DBPool, sender EmailSender, baseURL, senderFrom string, sessionTTL, refreshTTL, linkExpiry time.Duration) *AuthService {
	return &AuthService{
		pool:        pool,
		emailSender: sender,
		baseURL:     baseURL,
		senderFrom:  senderFrom,
		sessionTTL:  sessionTTL,
		refreshTTL:  refreshTTL,
		linkExpiry:  linkExpiry,
	}
}

// AuthResult is returned on successful authentication.
type AuthResult struct {
	User         *types.User         `json:"user"`
	SessionToken string              `json:"session_token"`
	RefreshToken string              `json:"refresh_token"`
	Session      *types.Session      `json:"session"`
	RefreshRT    *types.RefreshToken `json:"refresh_token_record"`
}

// RequestMagicLink generates a magic link for the given email.
// If the user doesn't exist, creates them with a bootstrap account.
func (s *AuthService) RequestMagicLink(ctx context.Context, email string) (rawToken string, err error) {
	// Validate email format
	_, err = mail.ParseAddress(email)
	if err != nil {
		return "", fmt.Errorf("invalid email address: %w", err)
	}
	email = strings.ToLower(strings.TrimSpace(email))

	// Ensure bootstrap account exists
	bootstrapAccount, err := s.ensureBootstrapAccount(ctx)
	if err != nil {
		return "", fmt.Errorf("ensure bootstrap account: %w", err)
	}

	// Find or create user (use transaction to prevent duplicates under concurrency)
	user, err := func() (*types.User, error) {
		tx, terr := s.pool.DB().BeginTx(ctx, nil)
		if terr != nil {
			return nil, fmt.Errorf("begin tx: %w", terr)
		}
		defer func() {
			_ = tx.Rollback()
		}()

		var user *types.User
		// Check under transaction lock
		rows, terr := tx.QueryContext(ctx, "SELECT id, account_id, email, name, role_id, status, created_at, updated_at, deleted_at FROM users WHERE email = ? AND deleted_at IS NULL", email)
		if terr == nil {
			results, _ := scanUsers(rows)
			if len(results) > 0 {
				user = results[0]
			}
		}

		var justCreated bool
		if user == nil {
			// Create new user. Use the transaction directly for role lookup
			// to avoid deadlock with SetMaxOpenConns(1).
			var userRoleID string
			terr = tx.QueryRowContext(ctx, "SELECT id FROM roles WHERE name = ?", "user").Scan(&userRoleID)
			if terr != nil {
				return nil, fmt.Errorf("get user role: %w", terr)
			}

			now := time.Now().UTC()
			user = &types.User{
				ID:        GenerateID(),
				AccountID: bootstrapAccount.ID,
				Email:     email,
				Name:      extractName(email),
				RoleID:    userRoleID,
				Status:    "active",
				CreatedAt: now,
				UpdatedAt: now,
			}

			_, terr = tx.ExecContext(ctx,
				"INSERT INTO users (id, account_id, email, name, role_id, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
				user.ID, user.AccountID, user.Email, user.Name, user.RoleID, user.Status, user.CreatedAt.Format(time.RFC3339), user.UpdatedAt.Format(time.RFC3339),
			)
			if terr != nil {
				// UNIQUE conflict: concurrent caller created the user first — fetch it.
				if isUniqueError(terr) {
					rows2, selErr := tx.QueryContext(ctx,
						"SELECT id, account_id, email, name, role_id, status, created_at, updated_at, deleted_at FROM users WHERE email = ? AND deleted_at IS NULL", email)
					if selErr == nil {
						results2, _ := scanUsers(rows2)
						if len(results2) > 0 {
							user = results2[0]
						}
					}
					if user == nil {
						return nil, fmt.Errorf("create user: UNIQUE conflict and could not find existing user: %w", terr)
					}
				} else {
					return nil, fmt.Errorf("create user: %w", terr)
				}
			} else {
				justCreated = true
			}
		}

		if terr := tx.Commit(); terr != nil {
			return nil, fmt.Errorf("commit tx: %w", terr)
		}

		// Write audit log AFTER commit (avoids holding the connection).
		if justCreated {
			s.writeAudit(ctx, user.ID, bootstrapAccount.ID, types.AuditEventOwnership, "auto_created_user", nil)
			log.Printf("[IDENTITY] Auto-created user %s (%s)", user.ID, email)
		}

		return user, nil
	}()
	if err != nil {
		return "", err
	}

	if user.Status != "active" {
		return "", fmt.Errorf("user account is not active: %s", user.Status)
	}

	// Generate magic link token (32 bytes hex = 64 chars)
	rawToken = generateRawToken(32)
	tokenHash := HashToken(rawToken)

	link := &types.MagicLink{
		ID:        GenerateID(),
		Email:     email,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(s.linkExpiry),
		CreatedAt: time.Now().UTC(),
	}
	if err := s.pool.MagicLinks.CreateMagicLink(ctx, link); err != nil {
		return "", fmt.Errorf("create magic link: %w", err)
	}

	// Construct magic link URL - use the implemented /api/v1 route
	magicLinkURL := s.baseURL + "/api/v1/auth/verify?token=" + rawToken

	// Send email
	subject := "Your TwelveReader Magic Link"
	body := fmt.Sprintf(`Welcome to TwelveReader!

Click the link below to sign in (valid for %d minutes):

%s

If you didn't request this, you can safely ignore this email.`, int(s.linkExpiry.Minutes()), magicLinkURL)

	if err := s.emailSender.SendMagicLink(email, subject, body); err != nil {
		return "", fmt.Errorf("send email: %w", err)
	}

	s.writeAudit(ctx, user.ID, bootstrapAccount.ID, types.AuditEventLoginSuccess, "magic_link_requested", map[string]string{"email": email})
	log.Printf("[IDENTITY] Magic link requested for %s", email)

	return rawToken, nil
}

// VerifyMagicLink verifies a magic link token and creates an authenticated session.
func (s *AuthService) VerifyMagicLink(ctx context.Context, rawToken, ipAddress, userAgent string) (*AuthResult, error) {
	tokenHash := HashToken(rawToken)

	// Atomic consume: verifies + marks used in one operation (race-safe)
	link, err := s.pool.MagicLinks.ConsumeMagicLink(ctx, tokenHash)
	if err != nil {
		s.writeAudit(ctx, "", "", types.AuditEventLoginFailed, "magic_link_consume_failed", map[string]string{"reason": err.Error()})
		return nil, fmt.Errorf("invalid or expired magic link")
	}

	// Find user by email
	user, err := s.pool.Users.GetUserByEmail(ctx, link.Email)
	if err != nil {
		return nil, fmt.Errorf("user not found for magic link")
	}

	if user.Status != "active" {
		s.writeAudit(ctx, user.ID, user.AccountID, types.AuditEventLoginFailed, "user_not_active", map[string]string{"email": user.Email})
		return nil, fmt.Errorf("user account is not active")
	}

	// Create session
	sessionToken := generateRawToken(32)
	session := &types.Session{
		ID:         GenerateID(),
		UserID:     user.ID,
		TokenHash:  HashToken(sessionToken),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		ExpiresAt:  time.Now().UTC().Add(s.sessionTTL),
		CreatedAt:  time.Now().UTC(),
		LastUsedAt: time.Now().UTC(),
	}

	// Create refresh token
	refreshRawToken := generateRawToken(32)
	refreshRT := &types.RefreshToken{
		ID:        GenerateID(),
		UserID:    user.ID,
		TokenHash: HashToken(refreshRawToken),
		IPAddress: ipAddress,
		UserAgent: userAgent,
		ExpiresAt: time.Now().UTC().Add(s.refreshTTL),
		CreatedAt: time.Now().UTC(),
	}

	// Store refresh token first so we can link it to the session
	if err := s.pool.RefreshTokens.CreateRefreshToken(ctx, refreshRT); err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
	}

	// Link session to its paired refresh token for atomic logout
	session.RefreshTokenID = refreshRT.ID
	if err := s.pool.Sessions.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Audit log
	s.writeAudit(ctx, user.ID, user.AccountID, types.AuditEventLoginSuccess, "magic_link_verified_login", map[string]string{
		"email":      user.Email,
		"session_id": session.ID,
	})
	s.writeAudit(ctx, user.ID, user.AccountID, types.AuditEventSessionCreate, "session_created", map[string]string{
		"session_id": session.ID,
	})

	log.Printf("[IDENTITY] User %s (%s) logged in via magic link", user.ID, user.Email)

	return &AuthResult{
		User:         user,
		SessionToken: sessionToken,
		RefreshToken: refreshRawToken,
		Session:      session,
		RefreshRT:    refreshRT,
	}, nil
}

// RefreshSession takes a refresh token and issues new session + refresh tokens.
func (s *AuthService) RefreshSession(ctx context.Context, rawRefreshToken, ipAddress, userAgent string) (*AuthResult, error) {
	tokenHash := HashToken(rawRefreshToken)

	// Atomic consume: verifies + marks used in one operation (race-safe)
	rt, err := s.pool.RefreshTokens.ConsumeRefreshToken(ctx, tokenHash)
	if err != nil {
		s.writeAudit(ctx, "", "", types.AuditEventLoginFailed, "refresh_token_consume_failed", map[string]string{"reason": err.Error()})
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Find user
	user, err := s.pool.Users.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if user.Status != "active" {
		return nil, fmt.Errorf("user account is not active")
	}

	// Create new session
	sessionToken := generateRawToken(32)
	session := &types.Session{
		ID:         GenerateID(),
		UserID:     user.ID,
		TokenHash:  HashToken(sessionToken),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		ExpiresAt:  time.Now().UTC().Add(s.sessionTTL),
		CreatedAt:  time.Now().UTC(),
		LastUsedAt: time.Now().UTC(),
	}

	// Create new refresh token (rotation)
	newRefreshRawToken := generateRawToken(32)
	newRefreshRT := &types.RefreshToken{
		ID:        GenerateID(),
		UserID:    user.ID,
		TokenHash: HashToken(newRefreshRawToken),
		IPAddress: ipAddress,
		UserAgent: userAgent,
		ExpiresAt: time.Now().UTC().Add(s.refreshTTL),
		CreatedAt: time.Now().UTC(),
	}

	// Store refresh token first so we can link it to the session
	if err := s.pool.RefreshTokens.CreateRefreshToken(ctx, newRefreshRT); err != nil {
		return nil, fmt.Errorf("create new refresh token: %w", err)
	}

	// Link session to its paired refresh token for atomic logout
	session.RefreshTokenID = newRefreshRT.ID
	if err := s.pool.Sessions.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create new session: %w", err)
	}

	s.writeAudit(ctx, user.ID, user.AccountID, types.AuditEventTokenRefresh, "session_refreshed", map[string]string{
		"new_session_id": session.ID,
	})

	log.Printf("[IDENTITY] User %s refreshed session", user.ID)

	return &AuthResult{
		User:         user,
		SessionToken: sessionToken,
		RefreshToken: newRefreshRawToken,
		Session:      session,
		RefreshRT:    newRefreshRT,
	}, nil
}

// Logout revokes only the current session and its associated refresh token.
// This allows other active sessions (e.g., on other devices) to remain valid.
// To revoke all sessions for a user, use RevokeAllUserSessions instead.
//
// Fail-closed: if refreshing the paired refresh token fails, Logout returns an error.
// For legacy sessions without a refresh_token_id (pre-migration 004), all active
// refresh tokens for that user are revoked as a safety measure.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	// Get session first to find user and paired refresh token for audit
	session, err := s.pool.Sessions.GetSessionByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found")
	}

	userID := session.UserID

	// Revoke the session
	if err := s.pool.Sessions.RevokeSession(ctx, sessionID); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	// Revoke paired refresh token (atomic logout for this device).
	// This prevents a stolen refresh token from creating new sessions after logout.
	if session.RefreshTokenID != "" {
		// Fail closed: if we can't revoke the paired refresh token, return error.
		if err := s.pool.RefreshTokens.RevokeRefreshToken(ctx, session.RefreshTokenID); err != nil {
			log.Printf("[IDENTITY] ERROR: failed to revoke paired refresh token %s on logout: %v", session.RefreshTokenID, err)
			s.writeAudit(ctx, userID, "", types.AuditEventSessionRevoke, "paired_refresh_token_revoke_FAILED_on_logout", map[string]string{
				"refresh_token_id": session.RefreshTokenID,
				"error":            err.Error(),
			})
			return fmt.Errorf("revoke paired refresh token: %w", err)
		}
		s.writeAudit(ctx, userID, "", types.AuditEventSessionRevoke, "paired_refresh_token_revoked_on_logout", map[string]string{
			"refresh_token_id": session.RefreshTokenID,
		})
	} else {
		// Legacy session (pre-migration 004): no paired refresh_token_id.
		// Revoke ALL active refresh tokens for this user to prevent stale tokens
		// from creating new sessions after logout.
		activeRTs, rtErr := s.pool.RefreshTokens.ListActiveByUser(ctx, userID)
		if rtErr == nil && len(activeRTs) > 0 {
			for _, rt := range activeRTs {
				_ = s.pool.RefreshTokens.RevokeRefreshToken(ctx, rt.ID)
			}
			s.writeAudit(ctx, userID, "", types.AuditEventSessionRevoke, "legacy_logout_revoked_all_user_refresh_tokens", map[string]string{
				"count": fmt.Sprintf("%d", len(activeRTs)),
			})
			log.Printf("[IDENTITY] Legacy logout for user %s: revoked %d active refresh tokens", userID, len(activeRTs))
		} else if rtErr != nil {
			log.Printf("[IDENTITY] Warning: failed to list active refresh tokens for legacy logout: %v", rtErr)
		}
	}

	s.writeAudit(ctx, userID, "", types.AuditEventLogout, "user_logged_out", map[string]string{
		"session_id": sessionID,
	})
	s.writeAudit(ctx, userID, "", types.AuditEventSessionRevoke, "session_revoked_on_logout", map[string]string{
		"session_id": sessionID,
	})

	log.Printf("[IDENTITY] User %s logged out (session %s, refresh_token %s)", userID, sessionID, session.RefreshTokenID)
	return nil
}

// RevokeAllUserSessions revokes ALL sessions and their paired refresh tokens for a user.
// This is the explicit "log out everywhere" operation — it cleanly removes both sessions
// and refresh tokens together, avoiding the inconsistent state where revoked sessions
// leave orphaned refresh tokens that can create new sessions.
func (s *AuthService) RevokeAllUserSessions(ctx context.Context, userID string) error {
	// Get all active sessions for this user
	sessions, err := s.pool.Sessions.ListActiveSessionsByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list active sessions: %w", err)
	}

	for _, sess := range sessions {
		_ = s.pool.Sessions.RevokeSession(ctx, sess.ID)
		// Revoke paired refresh token too
		if sess.RefreshTokenID != "" {
			_ = s.pool.RefreshTokens.RevokeRefreshToken(ctx, sess.RefreshTokenID)
		}
	}

	s.writeAudit(ctx, userID, "", types.AuditEventSessionRevoke, "all_user_sessions_revoked", map[string]string{
		"count": fmt.Sprintf("%d", len(sessions)),
	})

	log.Printf("[IDENTITY] Revoked %d sessions for user %s", len(sessions), userID)
	return nil
}

// GetSessionByTokenHash looks up a session by its token hash.
func (s *AuthService) GetSessionByTokenHash(ctx context.Context, rawSessionToken string) (*types.Session, error) {
	tokenHash := HashToken(rawSessionToken)
	session, err := s.pool.Sessions.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	// Check revoked
	if session.Revoked {
		return nil, fmt.Errorf("session revoked")
	}

	// Check expiry
	if time.Now().UTC().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// Update last used
	_, err = s.pool.DB().ExecContext(ctx, "UPDATE sessions SET last_used_at = ? WHERE id = ?", time.Now().UTC().Format(time.RFC3339), session.ID)
	if err != nil {
		log.Printf("[IDENTITY] Warning: failed to update session last_used_at: %v", err)
	}

	return session, nil
}

// GetUserByID returns a user by ID.
func (s *AuthService) GetUserByID(ctx context.Context, id string) (*types.User, error) {
	return s.pool.Users.GetUserByID(ctx, id)
}

// ensureBootstrapAccount ensures the bootstrap/default account and system roles exist.
// Concurrency-safe: handles duplicate key errors from concurrent callers by fetching existing record.
func (s *AuthService) ensureBootstrapAccount(ctx context.Context) (*types.Account, error) {
	// Check for bootstrap account by slug
	account, err := s.pool.Accounts.GetAccountBySlug(ctx, "bootstrap")
	if err == nil && account != nil {
		// Account exists — still ensure roles are present.
		s.ensureSystemRoles(ctx)
		return account, nil
	}

	// Create bootstrap account
	now := time.Now().UTC()
	account = &types.Account{
		ID:        GenerateID(),
		Name:      "Default",
		Slug:      "bootstrap",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.pool.Accounts.CreateAccount(ctx, account); err != nil {
		// Concurrent caller may have created it first — fetch existing on conflict.
		if isUniqueError(err) {
			existing, lookErr := s.pool.Accounts.GetAccountBySlug(ctx, "bootstrap")
			if lookErr != nil || existing == nil {
				return nil, fmt.Errorf("create bootstrap account (conflict): %w", err)
			}
			s.ensureSystemRoles(ctx)
			return existing, nil
		}
		return nil, fmt.Errorf("create bootstrap account: %w", err)
	}
	log.Printf("[IDENTITY] Created bootstrap account %s", account.ID)

	// Ensure system roles exist
	s.ensureSystemRoles(ctx)

	return account, nil
}

// ensureSystemRoles creates admin and user roles if they don't exist.
// Concurrency-safe: handles duplicate key errors from concurrent callers by fetching existing role.
func (s *AuthService) ensureSystemRoles(ctx context.Context) {
	roles := []struct {
		Name        string
		Description string
		IsSystem    bool
	}{
		{"admin", "Administrator with full access", true},
		{"user", "Regular user", true},
	}

	for _, r := range roles {
		existing, err := s.pool.Roles.GetRoleByName(ctx, r.Name)
		if err == nil && existing != nil {
			continue // Already exists
		}
		role := &types.Role{
			ID:          GenerateID(),
			Name:        r.Name,
			Description: r.Description,
			IsSystem:    r.IsSystem,
			CreatedAt:   time.Now().UTC(),
		}
		if err := s.pool.Roles.CreateRole(ctx, role); err != nil {
			// Concurrent caller may have created it — fetch existing on conflict.
			if isUniqueError(err) {
				existing, lookErr := s.pool.Roles.GetRoleByName(ctx, r.Name)
				if lookErr == nil && existing != nil {
					log.Printf("[IDENTITY] System role %s already exists (concurrent create)", r.Name)
					continue
				}
			}
			log.Printf("[IDENTITY] Warning: failed to create role %s: %v", r.Name, err)
		} else {
			log.Printf("[IDENTITY] Created system role: %s", r.Name)
		}
	}
}

func (s *AuthService) writeAudit(ctx context.Context, userID, accountID string, eventType types.AuditEventType, description string, metadata map[string]string) {
	entry := &types.AuditLogEntry{
		ID:          GenerateID(),
		UserID:      userID,
		AccountID:   accountID,
		EventType:   eventType,
		Description: description,
		Metadata:    metadata,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.pool.AuditLog.CreateEntry(ctx, entry); err != nil {
		log.Printf("[IDENTITY] Warning: failed to write audit log: %v", err)
	}
}

func generateRawToken(lengthBytes int) string {
	b := make([]byte, lengthBytes)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	return hex.EncodeToString(b)
}

func extractName(emailAddr string) string {
	addr, err := mail.ParseAddress(emailAddr)
	if err != nil {
		// Return local part as name fallback
		parts := strings.Split(emailAddr, "@")
		if len(parts) > 0 {
			return parts[0]
		}
		return emailAddr
	}
	if addr.Name != "" {
		return addr.Name
	}
	// Use local part of address
	parts := strings.Split(addr.Address, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return addr.Address
}

// isUniqueError checks if an error is a SQLite UNIQUE constraint violation.
// modernc.org/sqlite returns errors containing "UNIQUE constraint failed" or "unique".
func isUniqueError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique constraint failed") ||
		strings.Contains(msg, "is not unique")
}
