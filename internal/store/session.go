package store

import "database/sql"

// GetSessionValue returns the stored value for key, or "" if not set.
func (s *Store) GetSessionValue(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM session WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetSessionValue upserts key → value in the session table.
func (s *Store) SetSessionValue(key, value string) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO session (key, value) VALUES (?, ?)",
		key, value,
	)
	return err
}
