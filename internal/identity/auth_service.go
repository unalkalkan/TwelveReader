package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
type LogEmailSender struct{}

func (l *LogEmailSender) SendMagicLink(to, subject, body string) error {
	log.Printf("[EMAIL] To: %s, Subject: %s, Body:\n%s", to, subject, body)
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
	User         *types.User  `json:"user"`
	SessionToken string       `json:"session_token"`
	RefreshToken string       `json:"refresh_token"`
	Session      *types.Session     `json:"session"`
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

	// Find or create user
	user, err := s.pool.Users.GetUserByEmail(ctx, email)
	if err != nil {
		// User doesn't exist - find "user" role and create them
		userRole, err := s.pool.Roles.GetRoleByName(ctx, "user")
		if err != nil {
			return "", fmt.Errorf("get user role: %w", err)
		}

		user = &types.User{
			ID:        GenerateID(),
			AccountID: bootstrapAccount.ID,
			Email:     email,
			Name:      extractName(email),
			RoleID:    userRole.ID,
			Status:    "active",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		if err := s.pool.Users.CreateUser(ctx, user); err != nil {
			return "", fmt.Errorf("create user: %w", err)
		}

		s.writeAudit(ctx, user.ID, bootstrapAccount.ID, types.AuditEventOwnership, "auto_created_user", nil)
		log.Printf("[IDENTITY] Auto-created user %s (%s)", user.ID, email)
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

	// Construct magic link URL
	magicLinkURL := s.baseURL + "/auth/verify?token=" + rawToken

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

	link, err := s.pool.MagicLinks.GetMagicLinkByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired magic link")
	}

	// Check if already used
	if link.Used {
		s.writeAudit(ctx, "", "", types.AuditEventLoginFailed, "magic_link_already_used", map[string]string{"token_id": link.ID})
		return nil, fmt.Errorf("magic link already used")
	}

	// Check expiry
	if time.Now().UTC().After(link.ExpiresAt) {
		s.writeAudit(ctx, "", "", types.AuditEventLoginFailed, "magic_link_expired", map[string]string{"token_id": link.ID})
		return nil, fmt.Errorf("magic link expired")
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

	// Mark magic link as used
	if err := s.pool.MagicLinks.MarkUsed(ctx, link.ID); err != nil {
		log.Printf("[IDENTITY] Warning: failed to mark magic link used: %v", err)
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
	if err := s.pool.Sessions.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
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
	if err := s.pool.RefreshTokens.CreateRefreshToken(ctx, refreshRT); err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
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

	rt, err := s.pool.RefreshTokens.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	if rt.Revoked {
		s.writeAudit(ctx, rt.UserID, "", types.AuditEventLoginFailed, "refresh_token_revoked", map[string]string{"rt_id": rt.ID})
		return nil, fmt.Errorf("refresh token revoked")
	}

	if rt.Used {
		s.writeAudit(ctx, rt.UserID, "", types.AuditEventLoginFailed, "refresh_token_already_used", map[string]string{"rt_id": rt.ID})
		return nil, fmt.Errorf("refresh token already consumed")
	}

	if time.Now().UTC().After(rt.ExpiresAt) {
		s.writeAudit(ctx, rt.UserID, "", types.AuditEventLoginFailed, "refresh_token_expired", map[string]string{"rt_id": rt.ID})
		return nil, fmt.Errorf("refresh token expired")
	}

	// Find user
	user, err := s.pool.Users.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if user.Status != "active" {
		return nil, fmt.Errorf("user account is not active")
	}

	// Mark old refresh token as used (one-time use)
	_, err = s.pool.DB().ExecContext(ctx, "UPDATE refresh_tokens SET used = 1 WHERE id = ?", rt.ID)
	if err != nil {
		log.Printf("[IDENTITY] Warning: failed to mark refresh token used: %v", err)
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
	if err := s.pool.Sessions.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create new session: %w", err)
	}

	// Create new refresh token
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
	if err := s.pool.RefreshTokens.CreateRefreshToken(ctx, newRefreshRT); err != nil {
		return nil, fmt.Errorf("create new refresh token: %w", err)
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

// Logout revokes the current session and all associated refresh tokens.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	// Get session first to find user for audit
	session, err := s.pool.Sessions.GetSessionByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found")
	}

	if err := s.pool.Sessions.RevokeSession(ctx, sessionID); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	// Revoke all refresh tokens for this user
	// (to be safe - revoke all on logout)
	_, _ = s.pool.DB().ExecContext(ctx, "UPDATE refresh_tokens SET revoked = 1 WHERE user_id = ?", session.UserID)

	s.writeAudit(ctx, session.UserID, "", types.AuditEventLogout, "user_logged_out", map[string]string{
		"session_id": sessionID,
	})
	s.writeAudit(ctx, session.UserID, "", types.AuditEventSessionRevoke, "session_revoked_on_logout", map[string]string{
		"session_id": sessionID,
	})

	log.Printf("[IDENTITY] User %s logged out (session %s)", session.UserID, sessionID)
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
func (s *AuthService) ensureBootstrapAccount(ctx context.Context) (*types.Account, error) {
	// Check for bootstrap account by slug
	account, err := s.pool.Accounts.GetAccountBySlug(ctx, "bootstrap")
	if err == nil && account != nil {
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
		return nil, fmt.Errorf("create bootstrap account: %w", err)
	}
	log.Printf("[IDENTITY] Created bootstrap account %s", account.ID)

	// Ensure system roles exist
	s.ensureSystemRoles(ctx)

	return account, nil
}

// ensureSystemRoles creates admin and user roles if they don't exist.
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
