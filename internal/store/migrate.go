package store

import (
	"embed"
	"fmt"
	"sort"
	"strings"
)

// migrations holds all SQL files from internal/store/migrations at compile time.
// The path passed to go:embed is relative to the directory containing this file.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

// migrationSchema tracks which migrations have been applied.
const createMigrationTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    filename   TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);`

func (s *Store) Migrate() error {
	if _, err := s.db.Exec(createMigrationTable); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}

	// Sort by filename so 001_ always runs before 002_, etc.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		name := entry.Name()

		var count int
		err := s.db.QueryRow(
			"SELECT COUNT(*) FROM schema_migrations WHERE filename = ?", name,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if count > 0 {
			continue // already applied
		}

		sql, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", name, err)
		}

		if _, err := tx.Exec(string(sql)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}

		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (filename) VALUES (?)", name,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}

	return nil
}
