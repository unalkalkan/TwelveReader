package types

import "time"

// Account represents a SaaS account/workspace.
// V1 assumes one user per account; multi-user is a future extension.
type Account struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`   // URL-safe unique identifier
	Status    string     `json:"status"` // "active", "suspended", "deleted"
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// User represents an authenticated identity within an account.
type User struct {
	ID        string     `json:"id"`
	AccountID string     `json:"account_id"`
	Email     string     `json:"email"`
	Name      string     `json:"name,omitempty"`
	RoleID    string     `json:"role_id"`
	Status    string     `json:"status"` // "active", "pending", "suspended", "deleted"
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// Role defines a named permission set.
type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"` // "admin", "user"
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"` // System roles cannot be deleted
	CreatedAt   time.Time `json:"created_at"`
}

// Session represents an active authenticated session.
type Session struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	TokenHash      string    `json:"token_hash"` // SHA-256 of the actual session token
	IPAddress      string    `json:"ip_address,omitempty"`
	UserAgent      string    `json:"user_agent,omitempty"`
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
	LastUsedAt     time.Time `json:"last_used_at"`
	Revoked        bool      `json:"revoked"`
	RefreshTokenID string    `json:"refresh_token_id,omitempty"` // Paired refresh token for atomic logout
}

// RefreshToken represents a long-lived token for session renewal.
type RefreshToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"token_hash"` // SHA-256 of the actual refresh token
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `json:"revoked"`
	Used      bool      `json:"used"` // One-time use; set true after consumption
}

// MagicLink represents a one-time use email verification token.
type MagicLink struct {
	ID        string    `json:"id"`
	Email     string    `json:"-"` // Never expose in JSON
	TokenHash string    `json:"token_hash"`
	Used      bool      `json:"used"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// AuditEventType is a standard audit event category.
type AuditEventType string

const (
	AuditEventLoginSuccess  AuditEventType = "login_success"
	AuditEventLoginFailed   AuditEventType = "login_failed"
	AuditEventLogout        AuditEventType = "logout"
	AuditEventSessionCreate AuditEventType = "session_create"
	AuditEventSessionRevoke AuditEventType = "session_revoke"
	AuditEventTokenRefresh  AuditEventType = "token_refresh"
	AuditEventAdminAction   AuditEventType = "admin_action"
	AuditEventOwnership     AuditEventType = "ownership_change"
)

// AuditLogEntry records an auditable system event.
type AuditLogEntry struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id,omitempty"`
	AccountID   string            `json:"account_id,omitempty"`
	EventType   AuditEventType    `json:"event_type"`
	Description string            `json:"description"`
	IPAddress   string            `json:"ip_address,omitempty"`
	UserAgent   string            `json:"user_agent,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"` // Arbitrary key-value context
	CreatedAt   time.Time         `json:"created_at"`
}
