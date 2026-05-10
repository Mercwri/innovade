package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	_ "modernc.org/sqlite" // SQLite driver, no CGO required
)

const appName = "innovade"

type Store struct {
	db *sql.DB
}

// Linux:   $XDG_DATA_HOME/innovade/  (falls back to ~/.local/share/innovade/)
// macOS:   ~/Library/Application Support/innovade/
// Windows: %APPDATA%\innovade\
func New() (*Store, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, fmt.Errorf("resolve data dir: %w", err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir %s: %w", dir, err)
	}

	dbPath := filepath.Join(dir, appName+".db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db at %s: %w", dbPath, err)
	}

	// SQLite performs best with a single writer connection.
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.Migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

// Close releases the database connection. Call this on application shutdown.
func (s *Store) Close() error {
	return s.db.Close()
}

// DataDir returns the resolved user data directory for the application.
// Useful for displaying the DB path in a settings screen.
func DataDir() (string, error) {
	return dataDir()
}

func dataDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("APPDATA")
		if base == "" {
			return "", fmt.Errorf("%%APPDATA%% is not set")
		}
		return filepath.Join(base, appName), nil

	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", appName), nil

	default: // Linux and other Unix
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, appName), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", appName), nil
	}
}
