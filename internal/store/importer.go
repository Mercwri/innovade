package store

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Mercwri/innovade/internal/models"
)

// rawCard mirrors the JSON export structure exactly.
type rawCard struct {
	ID               string   `json:"id"`
	CardCode         string   `json:"card_code"`
	SetCode          string   `json:"set_code"`
	Name             string   `json:"name"`
	Category         string   `json:"category"`
	Rarity           string   `json:"rarity"`
	CreatedAt        string   `json:"created_at"`
	Color            []string `json:"color"`
	Cost             int      `json:"cost"`
	Level            *int     `json:"level"` // nullable in JSON
	Type             []string `json:"type"`
	AP               int      `json:"ap"`
	HP               int      `json:"hp"`
	Effect           *string  `json:"effect"` // nullable for tokens
	Burst            *string  `json:"burst"`  // nullable for tokens
	LinkRequirement  *string  `json:"link_requirement"`
	Set              string   `json:"set"`
	Locations        []string `json:"locations"`
	ImagePaths       []string `json:"_imagePaths"`
	DefaultImagePath string   `json:"_defaultImagePath"`
}

// ImportCardsFromJSON reads a JSON export file and upserts all cards into
// the database. Existing cards with the same card_code are updated in place.
func (s *Store) ImportCardsFromJSON(path string) (imported int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var raw []rawCard
	if err := json.NewDecoder(f).Decode(&raw); err != nil {
		return 0, fmt.Errorf("decode json: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Collect unique sets first
	setsSeen := map[string]rawCard{}
	for _, r := range raw {
		setsSeen[r.SetCode] = r
	}
	for code, r := range setsSeen {
		setType := string(models.SetTypeFromCode(code))
		_, err = tx.Exec(`
			INSERT INTO card_sets (code, name, set_type)
			VALUES (?, ?, ?)
			ON CONFLICT(code) DO UPDATE SET name=excluded.name, set_type=excluded.set_type
		`, code, r.Set, setType)
		if err != nil {
			return 0, fmt.Errorf("upsert set %s: %w", code, err)
		}
	}

	for _, r := range raw {
		card := mapRawCard(r)

		_, err = tx.Exec(`
			INSERT INTO cards
				(id, card_code, set_code, name, category, rarity, created_at,
				 cost, level, ap, hp, effect, burst, link_requirement, default_image_path)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT(card_code) DO UPDATE SET
				name=excluded.name,
				category=excluded.category,
				rarity=excluded.rarity,
				cost=excluded.cost,
				level=excluded.level,
				ap=excluded.ap,
				hp=excluded.hp,
				effect=excluded.effect,
				burst=excluded.burst,
				link_requirement=excluded.link_requirement,
				default_image_path=excluded.default_image_path
		`,
			card.ID, card.CardCode, card.SetCode, card.Name,
			string(card.Category), string(card.Rarity),
			card.CreatedAt.Format(time.RFC3339),
			card.Cost, card.Level, card.AP, card.HP,
			card.Effect, card.Burst, card.LinkRequirement,
			card.DefaultImagePath,
		)
		if err != nil {
			return 0, fmt.Errorf("upsert card %s: %w", card.CardCode, err)
		}

		// Replace child rows for this card
		for _, tbl := range []string{"card_colors", "card_types", "card_locations", "card_images"} {
			if _, err = tx.Exec("DELETE FROM "+tbl+" WHERE card_code=?", card.CardCode); err != nil {
				return 0, fmt.Errorf("clear %s for %s: %w", tbl, card.CardCode, err)
			}
		}

		for _, color := range card.Colors {
			if _, err = tx.Exec(
				"INSERT INTO card_colors (card_code, color) VALUES (?,?)",
				card.CardCode, string(color),
			); err != nil {
				return 0, fmt.Errorf("insert color for %s: %w", card.CardCode, err)
			}
		}

		for i, t := range card.Types {
			if _, err = tx.Exec(
				"INSERT INTO card_types (card_code, type, sort_order) VALUES (?,?,?)",
				card.CardCode, t, i,
			); err != nil {
				return 0, fmt.Errorf("insert type for %s: %w", card.CardCode, err)
			}
		}

		for _, loc := range card.Locations {
			if _, err = tx.Exec(
				"INSERT INTO card_locations (card_code, location) VALUES (?,?)",
				card.CardCode, string(loc),
			); err != nil {
				return 0, fmt.Errorf("insert location for %s: %w", card.CardCode, err)
			}
		}

		for i, imgPath := range card.ImagePaths {
			if _, err = tx.Exec(
				"INSERT INTO card_images (card_code, sort_order, image_path) VALUES (?,?,?)",
				card.CardCode, i, imgPath,
			); err != nil {
				return 0, fmt.Errorf("insert image for %s: %w", card.CardCode, err)
			}
		}

		imported++
	}

	return imported, tx.Commit()
}

// mapRawCard converts a rawCard from JSON into the clean domain model.
func mapRawCard(r rawCard) models.Card {
	c := models.Card{
		ID:               r.ID,
		CardCode:         r.CardCode,
		SetCode:          r.SetCode,
		SetName:          r.Set,
		Name:             r.Name,
		Category:         models.Category(r.Category),
		Rarity:           models.Rarity(r.Rarity),
		Cost:             r.Cost,
		AP:               r.AP,
		HP:               r.HP,
		DefaultImagePath: r.DefaultImagePath,
		ImagePaths:       r.ImagePaths,
	}

	// Nullable int: level
	if r.Level != nil {
		c.Level = *r.Level
	}

	// Nullable strings
	if r.Effect != nil {
		c.Effect = *r.Effect
	}
	if r.Burst != nil {
		c.Burst = *r.Burst
	}
	if r.LinkRequirement != nil {
		c.LinkRequirement = strings.TrimSpace(*r.LinkRequirement)
	}

	// Colors
	for _, col := range r.Color {
		c.Colors = append(c.Colors, models.Color(col))
	}

	// Types
	c.Types = r.Type

	// Locations
	for _, loc := range r.Locations {
		c.Locations = append(c.Locations, models.Location(loc))
	}

	// CreatedAt
	t, err := time.Parse(time.RFC3339Nano, r.CreatedAt)
	if err == nil {
		c.CreatedAt = t
	}

	return c
}
