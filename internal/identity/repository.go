package identity

import (
	"context"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// AccountRepository defines account CRUD operations.
type AccountRepository interface {
	CreateAccount(ctx context.Context, account *types.Account) error
	GetAccountByID(ctx context.Context, id string) (*types.Account, error)
	GetAccountBySlug(ctx context.Context, slug string) (*types.Account, error)
	ListAccounts(ctx context.Context) ([]*types.Account, error)
	UpdateAccount(ctx context.Context, account *types.Account) error
	DeleteAccount(ctx context.Context, id string) error
}

// UserRepository defines user CRUD operations.
type UserRepository interface {
	CreateUser(ctx context.Context, user *types.User) error
	GetUserByID(ctx context.Context, id string) (*types.User, error)
	GetUserByEmail(ctx context.Context, email string) (*types.User, error)
	ListUsersByAccount(ctx context.Context, accountID string) ([]*types.User, error)
	UpdateUser(ctx context.Context, user *types.User) error
	DeleteUser(ctx context.Context, id string) error
}

// RoleRepository defines role CRUD operations.
type RoleRepository interface {
	CreateRole(ctx context.Context, role *types.Role) error
	GetRoleByID(ctx context.Context, id string) (*types.Role, error)
	GetRoleByName(ctx context.Context, name string) (*types.Role, error)
	ListRoles(ctx context.Context) ([]*types.Role, error)
	UpdateRole(ctx context.Context, role *types.Role) error
	DeleteRole(ctx context.Context, id string) error
}

// SessionRepository defines session CRUD operations.
type SessionRepository interface {
	CreateSession(ctx context.Context, session *types.Session) error
	GetSessionByID(ctx context.Context, id string) (*types.Session, error)
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*types.Session, error)
	GetSessionByRefreshTokenID(ctx context.Context, refreshTokenID string) (*types.Session, error)
	ListActiveSessionsByUser(ctx context.Context, userID string) ([]*types.Session, error)
	RevokeSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) (int64, error)
}

// RefreshTokenRepository defines refresh token CRUD operations.
type RefreshTokenRepository interface {
	CreateRefreshToken(ctx context.Context, rt *types.RefreshToken) error
	GetRefreshTokenByID(ctx context.Context, id string) (*types.RefreshToken, error)
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*types.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id string) error
	ListActiveByUser(ctx context.Context, userID string) ([]*types.RefreshToken, error)
	ConsumeRefreshToken(ctx context.Context, tokenHash string) (*types.RefreshToken, error)
	DeleteExpiredTokens(ctx context.Context) (int64, error)
}

// AuditLogRepository defines audit log operations.
type AuditLogRepository interface {
	CreateEntry(ctx context.Context, entry *types.AuditLogEntry) error
	GetEntryByID(ctx context.Context, id string) (*types.AuditLogEntry, error)
	ListEntriesByUser(ctx context.Context, userID string, limit int) ([]*types.AuditLogEntry, error)
	ListEntriesByType(ctx context.Context, eventType types.AuditEventType, limit int) ([]*types.AuditLogEntry, error)
	ListRecentEntries(ctx context.Context, limit int) ([]*types.AuditLogEntry, error)
}

// MagicLinkRepository defines magic link operations.
type MagicLinkRepository interface {
	CreateMagicLink(ctx context.Context, ml *types.MagicLink) error
	GetMagicLinkByTokenHash(ctx context.Context, tokenHash string) (*types.MagicLink, error)
	MarkUsed(ctx context.Context, id string) error
	ConsumeMagicLink(ctx context.Context, tokenHash string) (*types.MagicLink, error)
	DeleteExpiredLinks(ctx context.Context) (int64, error)
}
