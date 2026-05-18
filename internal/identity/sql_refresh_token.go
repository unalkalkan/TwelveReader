package identity

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

type sqlRefreshTokenRepo struct{ db *sql.DB }

func (r *sqlRefreshTokenRepo) CreateRefreshToken(ctx context.Context, rt *types.RefreshToken) error {
	if rt.ID == "" {
		rt.ID = GenerateID()
	}
	now := time.Now().UTC()
	rt.CreatedAt = now
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO refresh_tokens (id, user_id, token_hash, ip_address, user_agent, expires_at, created_at, revoked, used) VALUES (?, ?, ?, ?, ?, ?, ?, 0, 0)",
		rt.ID, rt.UserID, rt.TokenHash, rt.IPAddress, rt.UserAgent, rt.ExpiresAt.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *sqlRefreshTokenRepo) GetRefreshTokenByID(ctx context.Context, id string) (*types.RefreshToken, error) {
	rt, err := r.getToken(ctx, "id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("get refresh token by id: %w", err)
	}
	return rt, nil
}

func (r *sqlRefreshTokenRepo) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*types.RefreshToken, error) {
	rt, err := r.getToken(ctx, "token_hash = ?", tokenHash)
	if err != nil {
		return nil, fmt.Errorf("get refresh token by hash: %w", err)
	}
	return rt, nil
}

func (r *sqlRefreshTokenRepo) RevokeRefreshToken(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "UPDATE refresh_tokens SET revoked = 1 WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke refresh token rows affected: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("revoke refresh token: no row updated (id=%s, may already be revoked or not found)", id)
	}
	return nil
}

// ListActiveByUser returns all active (not revoked, not expired) refresh tokens for a user.
func (r *sqlRefreshTokenRepo) ListActiveByUser(ctx context.Context, userID string) ([]*types.RefreshToken, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, user_id, token_hash, ip_address, user_agent, expires_at, created_at, revoked, used FROM refresh_tokens WHERE user_id = ? AND revoked = 0 AND expires_at > ?",
		userID, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("list active refresh tokens: %w", err)
	}
	defer rows.Close()
	return scanRefreshTokens(rows)
}

// ConsumeRefreshToken atomically verifies and marks a refresh token as used.
// Returns the RefreshToken if successful, or error if already used/revoked/expired/not found.
func (r *sqlRefreshTokenRepo) ConsumeRefreshToken(ctx context.Context, tokenHash string) (*types.RefreshToken, error) {
	// Atomic conditional update: only succeed if not used, not revoked, and not expired
	result, err := r.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET used = 1 WHERE token_hash = ? AND used = 0 AND revoked = 0 AND expires_at > ?",
		tokenHash, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("consume refresh token: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("consume refresh token rows affected: %w", err)
	}
	if affected != 1 {
		return nil, fmt.Errorf("refresh token invalid, already consumed, revoked, or expired")
	}

	// Fetch the consumed token (now used=1) to return details
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, user_id, token_hash, ip_address, user_agent, expires_at, created_at, revoked, used FROM refresh_tokens WHERE token_hash = ?",
		tokenHash,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch consumed refresh token: %w", err)
	}
	defer rows.Close()
	results, _ := scanRefreshTokens(rows)
	if len(results) == 0 {
		return nil, fmt.Errorf("refresh token not found after consume")
	}
	return results[0], nil
}

func (r *sqlRefreshTokenRepo) DeleteExpiredTokens(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM refresh_tokens WHERE expires_at < ? OR revoked = 1", time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("delete expired tokens: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

func (r *sqlRefreshTokenRepo) getToken(ctx context.Context, where string, arg any) (*types.RefreshToken, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, user_id, token_hash, ip_address, user_agent, expires_at, created_at, revoked, used FROM refresh_tokens WHERE "+where, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results, _ := scanRefreshTokens(rows)
	if len(results) == 0 {
		return nil, fmt.Errorf("refresh token not found")
	}
	return results[0], nil
}

func scanRefreshTokens(rows *sql.Rows) ([]*types.RefreshToken, error) {
	var tokens []*types.RefreshToken
	for rows.Next() {
		var rt types.RefreshToken
		var revoked, used int
		var expiresAt, createdAt string
		if err := rows.Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.IPAddress, &rt.UserAgent, &expiresAt, &createdAt, &revoked, &used); err != nil {
			return nil, fmt.Errorf("scan refresh token: %w", err)
		}
		rt.Revoked = revoked != 0
		rt.Used = used != 0
		rt.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		rt.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		tokens = append(tokens, &rt)
	}
	return tokens, rows.Err()
}
