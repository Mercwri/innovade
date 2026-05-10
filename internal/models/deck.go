package models

import "time"

type DeckEntry struct {
	CardCode string
	Quantity int
}

type Deck struct {
	ID          string
	Name        string
	Description string
	Entries     []DeckEntry // ordered as the player arranged them
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (d *Deck) TotalCards() int {
	total := 0
	for _, e := range d.Entries {
		total += e.Quantity
	}
	return total
}

func (d *Deck) CardCount(cardCode string) int {
	for _, e := range d.Entries {
		if e.CardCode == cardCode {
			return e.Quantity
		}
	}
	return 0
}

func (d *Deck) AddCard(cardCode string) int {
	for i, e := range d.Entries {
		if e.CardCode == cardCode {
			d.Entries[i].Quantity++
			return d.Entries[i].Quantity
		}
	}
	d.Entries = append(d.Entries, DeckEntry{CardCode: cardCode, Quantity: 1})
	return 1
}

func (d *Deck) RemoveCard(cardCode string) int {
	for i, e := range d.Entries {
		if e.CardCode == cardCode {
			if e.Quantity <= 1 {
				d.Entries = append(d.Entries[:i], d.Entries[i+1:]...)
				return 0
			}
			d.Entries[i].Quantity--
			return d.Entries[i].Quantity
		}
	}
	return 0
}

type ValidationError struct {
	Code    string // machine-readable code, e.g. "deck_too_small"
	Message string // human-readable description
}

type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

const (
	DeckMinSize      = 50
	DeckMaxSize      = 50
	MaxCopiesPerCard = 4
)
