package identity

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

type sqlSessionRepo struct{ db *sql.DB }

func (r *sqlSessionRepo) CreateSession(ctx context.Context, s *types.Session) error {
	if s.ID == "" {
		s.ID = GenerateID()
	}
	now := time.Now().UTC()
	s.CreatedAt = now
	s.LastUsedAt = now
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO sessions (id, user_id, token_hash, ip_address, user_agent, expires_at, created_at, last_used_at, revoked) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0)",
		s.ID, s.UserID, s.TokenHash, s.IPAddress, s.UserAgent, s.ExpiresAt.Format(time.RFC3339), s.CreatedAt.Format(time.RFC3339), s.LastUsedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *sqlSessionRepo) GetSessionByID(ctx context.Context, id string) (*types.Session, error) {
	s, err := r.getSession(ctx, "id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("get session by id: %w", err)
	}
	return s, nil
}

func (r *sqlSessionRepo) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*types.Session, error) {
	s, err := r.getSession(ctx, "token_hash = ?", tokenHash)
	if err != nil {
		return nil, fmt.Errorf("get session by token hash: %w", err)
	}
	return s, nil
}

func (r *sqlSessionRepo) ListActiveSessionsByUser(ctx context.Context, userID string) ([]*types.Session, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, user_id, token_hash, ip_address, user_agent, expires_at, created_at, last_used_at, revoked FROM sessions WHERE user_id = ? AND revoked = 0 AND expires_at > ?",
		userID, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("list active sessions: %w", err)
	}
	defer rows.Close()
	return scanSessions(rows)
}

func (r *sqlSessionRepo) RevokeSession(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE sessions SET revoked = 1 WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (r *sqlSessionRepo) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < ? OR revoked = 1", time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

func (r *sqlSessionRepo) getSession(ctx context.Context, where string, arg any) (*types.Session, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, user_id, token_hash, ip_address, user_agent, expires_at, created_at, last_used_at, revoked FROM sessions WHERE "+where, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results, _ := scanSessions(rows)
	if len(results) == 0 {
		return nil, fmt.Errorf("session not found")
	}
	return results[0], nil
}

func scanSessions(rows *sql.Rows) ([]*types.Session, error) {
	var sessions []*types.Session
	for rows.Next() {
		var s types.Session
		var revoked int
		var expiresAt, createdAt, lastUsedAt string
		if err := rows.Scan(&s.ID, &s.UserID, &s.TokenHash, &s.IPAddress, &s.UserAgent, &expiresAt, &createdAt, &lastUsedAt, &revoked); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		s.Revoked = revoked != 0
		s.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		s.LastUsedAt, _ = time.Parse(time.RFC3339, lastUsedAt)
		sessions = append(sessions, &s)
	}
	return sessions, rows.Err()
}
