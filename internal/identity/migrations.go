package identity

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type migrationStep struct {
	version int
	name    string
	sql     []byte
}

func loadBundledMigrations() []migrationStep {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil
	}

	var steps []migrationStep
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := migrationFS.ReadFile(filepath.Join("migrations", entry.Name()))
		if err != nil {
			continue
		}
		var version int
		n, _ := fmt.Sscanf(entry.Name(), "%d_", &version)
		if n != 1 {
			continue
		}
		steps = append(steps, migrationStep{
			version: version,
			name:    entry.Name(),
			sql:     data,
		})
	}

	sort.Slice(steps, func(i, j int) bool {
		return steps[i].version < steps[j].version
	})
	return steps
}

func (p *DBPool) runMigrations() error {
	typeInfoTable := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		)`
	if _, err := p.db.Exec(typeInfoTable); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied := make(map[int]bool)
	rows, err := p.db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("query migrations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err == nil {
			applied[v] = true
		}
	}

	now := formatTimeUTC()
	for _, step := range p.migrations {
		if applied[step.version] {
			continue
		}

		tx, err := p.db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", step.name, err)
		}

		if _, err := tx.Exec(string(step.sql)); err != nil {
			tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", step.name, err)
		}

		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
			step.version, now,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", step.name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", step.name, err)
		}
	}

	return nil
}
