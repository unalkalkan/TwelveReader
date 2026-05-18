package identity

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

type sqlRoleRepo struct{ db *sql.DB }

func (r *sqlRoleRepo) CreateRole(ctx context.Context, role *types.Role) error {
	if role.ID == "" {
		role.ID = GenerateID()
	}
	role.CreatedAt = time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO roles (id, name, description, is_system, created_at) VALUES (?, ?, ?, ?, ?)",
		role.ID, role.Name, role.Description, role.IsSystem, role.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create role: %w", err)
	}
	return nil
}

func (r *sqlRoleRepo) GetRoleByID(ctx context.Context, id string) (*types.Role, error) {
	role, err := r.getRole(ctx, "id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("get role by id: %w", err)
	}
	return role, nil
}

func (r *sqlRoleRepo) GetRoleByName(ctx context.Context, name string) (*types.Role, error) {
	role, err := r.getRole(ctx, "name = ?", name)
	if err != nil {
		return nil, fmt.Errorf("get role by name: %w", err)
	}
	return role, nil
}

func (r *sqlRoleRepo) ListRoles(ctx context.Context) ([]*types.Role, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, description, is_system, created_at FROM roles")
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()
	return scanRoles(rows)
}

func (r *sqlRoleRepo) UpdateRole(ctx context.Context, role *types.Role) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE roles SET name=?, description=? WHERE id=?",
		role.Name, role.Description, role.ID,
	)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	return nil
}

func (r *sqlRoleRepo) DeleteRole(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM roles WHERE id = ? AND is_system = 0", id)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("role not found or is system role")
	}
	return nil
}

func (r *sqlRoleRepo) getRole(ctx context.Context, where string, arg any) (*types.Role, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, description, is_system, created_at FROM roles WHERE "+where, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results, _ := scanRoles(rows)
	if len(results) == 0 {
		return nil, fmt.Errorf("role not found")
	}
	return results[0], nil
}

func scanRoles(rows *sql.Rows) ([]*types.Role, error) {
	var roles []*types.Role
	for rows.Next() {
		var r types.Role
		var isSystem int
		var createdAt string
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &isSystem, &createdAt); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		r.IsSystem = isSystem != 0
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		roles = append(roles, &r)
	}
	return roles, rows.Err()
}
