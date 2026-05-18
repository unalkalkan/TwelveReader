package identity

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

type sqlMagicLinkRepo struct{ db *sql.DB }

func (r *sqlMagicLinkRepo) CreateMagicLink(ctx context.Context, ml *types.MagicLink) error {
	if ml.ID == "" {
		ml.ID = GenerateID()
	}
	ml.CreatedAt = time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO magic_links (id, email, token_hash, used, expires_at, created_at) VALUES (?, ?, ?, 0, ?, ?)",
		ml.ID, ml.Email, ml.TokenHash, ml.ExpiresAt.Format(time.RFC3339), ml.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create magic link: %w", err)
	}
	return nil
}

func (r *sqlMagicLinkRepo) GetMagicLinkByTokenHash(ctx context.Context, tokenHash string) (*types.MagicLink, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, email, token_hash, used, expires_at, created_at FROM magic_links WHERE token_hash = ?",
		tokenHash,
	)
	if err != nil {
		return nil, fmt.Errorf("get magic link by token hash: %w", err)
	}
	defer rows.Close()
	results, _ := scanMagicLinks(rows)
	if len(results) == 0 {
		return nil, fmt.Errorf("magic link not found")
	}
	return results[0], nil
}

func (r *sqlMagicLinkRepo) MarkUsed(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE magic_links SET used = 1 WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("mark magic link used: %w", err)
	}
	return nil
}

// ConsumeMagicLink atomically verifies and marks a magic link as used.
// Returns the MagicLink if successful, or error if already used/expired/not found.
func (r *sqlMagicLinkRepo) ConsumeMagicLink(ctx context.Context, tokenHash string) (*types.MagicLink, error) {
	// Atomic conditional update: only succeed if not used and not expired
	result, err := r.db.ExecContext(ctx,
		"UPDATE magic_links SET used = 1 WHERE token_hash = ? AND used = 0 AND expires_at > ?",
		tokenHash, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("consume magic link: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("consume magic link rows affected: %w", err)
	}
	if affected != 1 {
		return nil, fmt.Errorf("magic link invalid, already used, or expired")
	}

	// Fetch the consumed link (now used=1) to return email
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, email, token_hash, used, expires_at, created_at FROM magic_links WHERE token_hash = ?",
		tokenHash,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch consumed magic link: %w", err)
	}
	defer rows.Close()
	results, _ := scanMagicLinks(rows)
	if len(results) == 0 {
		return nil, fmt.Errorf("magic link not found after consume")
	}
	return results[0], nil
}

func (r *sqlMagicLinkRepo) DeleteExpiredLinks(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		"DELETE FROM magic_links WHERE expires_at < ? OR used = 1",
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("delete expired magic links: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

func scanMagicLinks(rows *sql.Rows) ([]*types.MagicLink, error) {
	var links []*types.MagicLink
	for rows.Next() {
		var ml types.MagicLink
		var used int
		var expiresAt, createdAt string
		if err := rows.Scan(&ml.ID, &ml.Email, &ml.TokenHash, &used, &expiresAt, &createdAt); err != nil {
			return nil, fmt.Errorf("scan magic link: %w", err)
		}
		ml.Used = used != 0
		ml.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		ml.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		links = append(links, &ml)
	}
	return links, rows.Err()
}
