package identity

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

type sqlAccountRepo struct{ db *sql.DB }

func (r *sqlAccountRepo) CreateAccount(ctx context.Context, a *types.Account) error {
	if a.ID == "" {
		a.ID = GenerateID()
	}
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	if a.Status == "" {
		a.Status = "active"
	}
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO accounts (id, name, slug, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		a.ID, a.Name, a.Slug, a.Status, a.CreatedAt.Format(time.RFC3339), a.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}
	return nil
}

func (r *sqlAccountRepo) GetAccountByID(ctx context.Context, id string) (*types.Account, error) {
	a, err := r.getAccount(ctx, "id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("get account by id: %w", err)
	}
	return a, nil
}

func (r *sqlAccountRepo) GetAccountBySlug(ctx context.Context, slug string) (*types.Account, error) {
	a, err := r.getAccount(ctx, "slug = ?", slug)
	if err != nil {
		return nil, fmt.Errorf("get account by slug: %w", err)
	}
	return a, nil
}

func (r *sqlAccountRepo) ListAccounts(ctx context.Context) ([]*types.Account, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, slug, status, created_at, updated_at, deleted_at FROM accounts WHERE deleted_at IS NULL")
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()
	return scanAccounts(rows)
}

func (r *sqlAccountRepo) UpdateAccount(ctx context.Context, a *types.Account) error {
	a.UpdatedAt = time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		"UPDATE accounts SET name=?, slug=?, status=?, updated_at=? WHERE id=?",
		a.Name, a.Slug, a.Status, a.UpdatedAt.Format(time.RFC3339), a.ID,
	)
	if err != nil {
		return fmt.Errorf("update account: %w", err)
	}
	return nil
}

func (r *sqlAccountRepo) DeleteAccount(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, "UPDATE accounts SET deleted_at=?, updated_at=? WHERE id=?", now.Format(time.RFC3339), now.Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	return nil
}

func (r *sqlAccountRepo) getAccount(ctx context.Context, where string, arg any) (*types.Account, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, slug, status, created_at, updated_at, deleted_at FROM accounts WHERE "+where, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results, err := scanAccounts(rows)
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("account not found")
	}
	return results[0], nil
}

func scanAccounts(rows *sql.Rows) ([]*types.Account, error) {
	var accounts []*types.Account
	for rows.Next() {
		var a types.Account
		var deletedAt sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&a.ID, &a.Name, &a.Slug, &a.Status, &createdAt, &updatedAt, &deletedAt); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if deletedAt.Valid && deletedAt.String != "" {
			t, _ := time.Parse(time.RFC3339, deletedAt.String)
			a.DeletedAt = &t
		}
		accounts = append(accounts, &a)
	}
	return accounts, rows.Err()
}
