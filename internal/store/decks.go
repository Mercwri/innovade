package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Mercwri/innovade/internal/models"
)

// ListDecks returns all saved decks ordered by most recently updated.
func (s *Store) ListDecks() ([]models.Deck, error) {
	rows, err := s.db.Query(`
		SELECT id, name, description, created_at, updated_at
		FROM decks ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list decks: %w", err)
	}
	defer rows.Close()

	var decks []models.Deck
	for rows.Next() {
		d, err := scanDeck(rows)
		if err != nil {
			return nil, err
		}
		decks = append(decks, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range decks {
		if err := s.hydrateDeck(&decks[i]); err != nil {
			return nil, err
		}
	}
	return decks, nil
}

// LoadDeck returns a single deck by ID with all its entries.
// Returns sql.ErrNoRows if not found.
func (s *Store) LoadDeck(id string) (*models.Deck, error) {
	row := s.db.QueryRow(`
		SELECT id, name, description, created_at, updated_at
		FROM decks WHERE id = ?
	`, id)

	d, err := scanDeck(row)
	if err != nil {
		return nil, err
	}
	if err := s.hydrateDeck(d); err != nil {
		return nil, err
	}
	return d, nil
}

// SaveDeck creates a new deck or updates an existing one (matched by ID).
// If d.ID is empty, a new UUID is assigned and the deck is inserted.
func (s *Store) SaveDeck(d *models.Deck) error {
	now := time.Now().UTC()

	if d.ID == "" {
		d.ID = uuid.NewString()
		d.CreatedAt = now
	}
	d.UpdatedAt = now

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.Exec(`
		INSERT INTO decks (id, name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name        = excluded.name,
			description = excluded.description,
			updated_at  = excluded.updated_at
	`, d.ID, d.Name, d.Description,
		d.CreatedAt.Format(time.RFC3339),
		d.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("upsert deck: %w", err)
	}

	// Replace all entries — simplest correct approach for a 50-card deck.
	if _, err = tx.Exec("DELETE FROM deck_entries WHERE deck_id = ?", d.ID); err != nil {
		return fmt.Errorf("clear entries: %w", err)
	}

	for i, entry := range d.Entries {
		_, err = tx.Exec(`
			INSERT INTO deck_entries (deck_id, card_code, quantity, sort_order)
			VALUES (?, ?, ?, ?)
		`, d.ID, entry.CardCode, entry.Quantity, i)
		if err != nil {
			return fmt.Errorf("insert entry %s: %w", entry.CardCode, err)
		}
	}

	return tx.Commit()
}

// DeleteDeck removes a deck and all its entries by ID.
// Returns sql.ErrNoRows if the deck does not exist.
func (s *Store) DeleteDeck(id string) error {
	res, err := s.db.Exec("DELETE FROM decks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete deck: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// --- helpers ---

func scanDeck(row scanner) (*models.Deck, error) {
	var d models.Deck
	var createdAt, updatedAt string
	err := row.Scan(&d.ID, &d.Name, &d.Description, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("scan deck: %w", err)
	}
	d.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	d.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &d, nil
}

func (s *Store) hydrateDeck(d *models.Deck) error {
	rows, err := s.db.Query(`
		SELECT card_code, quantity
		FROM deck_entries
		WHERE deck_id = ?
		ORDER BY sort_order
	`, d.ID)
	if err != nil {
		return fmt.Errorf("load entries for deck %s: %w", d.ID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var entry models.DeckEntry
		if err := rows.Scan(&entry.CardCode, &entry.Quantity); err != nil {
			return err
		}
		d.Entries = append(d.Entries, entry)
	}
	return rows.Err()
}
