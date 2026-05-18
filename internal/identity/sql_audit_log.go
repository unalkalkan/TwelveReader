package identity

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

type sqlAuditLogRepo struct{ db *sql.DB }

func (r *sqlAuditLogRepo) CreateEntry(ctx context.Context, entry *types.AuditLogEntry) error {
	if entry.ID == "" {
		entry.ID = GenerateID()
	}
	now := time.Now().UTC()
	entry.CreatedAt = now

	metaJSON, _ := marshalJSONMetadata(entry.Metadata)

	var userID, accountID sql.NullString
	if entry.UserID != "" {
		userID = sql.NullString{String: entry.UserID, Valid: true}
	}
	if entry.AccountID != "" {
		accountID = sql.NullString{String: entry.AccountID, Valid: true}
	}

	_, err := r.db.ExecContext(ctx,
		"INSERT INTO audit_log (id, user_id, account_id, event_type, description, ip_address, user_agent, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		entry.ID, userID, accountID, entry.EventType, entry.Description, entry.IPAddress, entry.UserAgent, string(metaJSON), now.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("create audit log entry: %w", err)
	}
	return nil
}

func (r *sqlAuditLogRepo) GetEntryByID(ctx context.Context, id string) (*types.AuditLogEntry, error) {
	entry, err := r.getEntry(ctx, "id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("get audit log entry: %w", err)
	}
	return entry, nil
}

func (r *sqlAuditLogRepo) ListEntriesByUser(ctx context.Context, userID string, limit int) ([]*types.AuditLogEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, user_id, account_id, event_type, description, ip_address, user_agent, metadata, created_at FROM audit_log WHERE user_id = ? ORDER BY created_at DESC LIMIT ?",
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit log by user: %w", err)
	}
	defer rows.Close()
	return scanAuditEntries(rows)
}

func (r *sqlAuditLogRepo) ListEntriesByType(ctx context.Context, eventType types.AuditEventType, limit int) ([]*types.AuditLogEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, user_id, account_id, event_type, description, ip_address, user_agent, metadata, created_at FROM audit_log WHERE event_type = ? ORDER BY created_at DESC LIMIT ?",
		string(eventType), limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit log by type: %w", err)
	}
	defer rows.Close()
	return scanAuditEntries(rows)
}

func (r *sqlAuditLogRepo) ListRecentEntries(ctx context.Context, limit int) ([]*types.AuditLogEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, user_id, account_id, event_type, description, ip_address, user_agent, metadata, created_at FROM audit_log ORDER BY created_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list recent audit log: %w", err)
	}
	defer rows.Close()
	return scanAuditEntries(rows)
}

func (r *sqlAuditLogRepo) getEntry(ctx context.Context, where string, arg any) (*types.AuditLogEntry, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, user_id, account_id, event_type, description, ip_address, user_agent, metadata, created_at FROM audit_log WHERE "+where, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results, _ := scanAuditEntries(rows)
	if len(results) == 0 {
		return nil, fmt.Errorf("audit log entry not found")
	}
	return results[0], nil
}

func scanAuditEntries(rows *sql.Rows) ([]*types.AuditLogEntry, error) {
	var entries []*types.AuditLogEntry
	for rows.Next() {
		var e types.AuditLogEntry
		var userID, accountID sql.NullString
		var metaBytes []byte
		var createdAt string
		if err := rows.Scan(&e.ID, &userID, &accountID, &e.EventType, &e.Description, &e.IPAddress, &e.UserAgent, &metaBytes, &createdAt); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		if userID.Valid {
			e.UserID = userID.String
		}
		if accountID.Valid {
			e.AccountID = accountID.String
		}
		m, _ := parseJSONMetadata(metaBytes)
		e.Metadata = m
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}
