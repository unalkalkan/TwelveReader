package identity

import (
	"database/sql"
	"fmt"
)

// DBPool wraps a *sql.DB and exposes the sub-repositories.
type DBPool struct {
	db         *sql.DB
	migrations []migrationStep

	Accounts      AccountRepository
	Users         UserRepository
	Roles         RoleRepository
	Sessions      SessionRepository
	RefreshTokens RefreshTokenRepository
	AuditLog      AuditLogRepository
	MagicLinks    MagicLinkRepository
}

// NewDBPool opens an SQLite database at the given path, runs pending migrations,
// and returns a ready-to-use pool with all identity repositories wired.
func NewDBPool(dbPath string) (*DBPool, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Serialize writes: SQLite handles concurrent reads fine but not writes.
	// MaxOpenConns(1) prevents SQLITE_BUSY errors from multiple writers.
	db.SetMaxOpenConns(1)

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("pragma wal: %w", err)
	}

	// Busy timeout: if another connection holds a write lock, wait up to 5s
	// instead of returning SQLITE_BUSY immediately.
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("pragma busy_timeout: %w", err)
	}

	// Enable foreign keys.
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("pragma fk: %w", err)
	}

	pool := &DBPool{
		db:         db,
		migrations: loadBundledMigrations(),
	}

	if err := pool.runMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	pool.Accounts = &sqlAccountRepo{db}
	pool.Users = &sqlUserRepo{db}
	pool.Roles = &sqlRoleRepo{db}
	pool.Sessions = &sqlSessionRepo{db}
	pool.RefreshTokens = &sqlRefreshTokenRepo{db}
	pool.AuditLog = &sqlAuditLogRepo{db}
	pool.MagicLinks = &sqlMagicLinkRepo{db}

	return pool, nil
}

// Close closes the underlying database connection.
func (p *DBPool) Close() error {
	return p.db.Close()
}

// DB returns the underlying *sql.DB for advanced usage.
func (p *DBPool) DB() *sql.DB {
	return p.db
}
