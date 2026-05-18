package identity

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

type sqlUserRepo struct{ db *sql.DB }

func (r *sqlUserRepo) CreateUser(ctx context.Context, u *types.User) error {
	if u.ID == "" {
		u.ID = GenerateID()
	}
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now
	if u.Status == "" {
		u.Status = "active"
	}
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO users (id, account_id, email, name, role_id, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		u.ID, u.AccountID, u.Email, u.Name, u.RoleID, u.Status, u.CreatedAt.Format(time.RFC3339), u.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *sqlUserRepo) GetUserByID(ctx context.Context, id string) (*types.User, error) {
	u, err := r.getUser(ctx, "id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (r *sqlUserRepo) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	u, err := r.getUser(ctx, "email = ? AND deleted_at IS NULL", email)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func (r *sqlUserRepo) ListUsersByAccount(ctx context.Context, accountID string) ([]*types.User, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, account_id, email, name, role_id, status, created_at, updated_at, deleted_at FROM users WHERE account_id = ? AND deleted_at IS NULL", accountID)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	return scanUsers(rows)
}

func (r *sqlUserRepo) UpdateUser(ctx context.Context, u *types.User) error {
	u.UpdatedAt = time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		"UPDATE users SET name=?, role_id=?, status=?, updated_at=? WHERE id=?",
		u.Name, u.RoleID, u.Status, u.UpdatedAt.Format(time.RFC3339), u.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *sqlUserRepo) DeleteUser(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, "UPDATE users SET deleted_at=?, status='deleted', updated_at=? WHERE id=?", now.Format(time.RFC3339), now.Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (r *sqlUserRepo) getUser(ctx context.Context, where string, arg any) (*types.User, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, account_id, email, name, role_id, status, created_at, updated_at, deleted_at FROM users WHERE "+where, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results, err := scanUsers(rows)
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	return results[0], nil
}

func scanUsers(rows *sql.Rows) ([]*types.User, error) {
	var users []*types.User
	for rows.Next() {
		var u types.User
		var deletedAt sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&u.ID, &u.AccountID, &u.Email, &u.Name, &u.RoleID, &u.Status, &createdAt, &updatedAt, &deletedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if deletedAt.Valid && deletedAt.String != "" {
			t, _ := time.Parse(time.RFC3339, deletedAt.String)
			u.DeletedAt = &t
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}
