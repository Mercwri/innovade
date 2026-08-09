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

	pilotNames := collectPilotNames(raw)
	typeCanon := collectCanonicalTypes(raw)

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
		card.Types = normalizeTypes(r.Type, typeCanon)
		card.LinkRequirement = fixLinkRequirement(card.LinkRequirement, pilotNames)

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

// collectPilotNames scans the full import for every name that can appear on
// the pilot side of a link requirement: Pilot cards by name, and Command
// cards that embed a 【Pilot】[Name] marker in their effect text. The source
// export sometimes wraps one of these names in "(...)" or leaves it bare
// instead of using "[...]", which fixLinkRequirement uses this set to detect
// and correct.
func collectPilotNames(raw []rawCard) map[string]bool {
	names := make(map[string]bool)
	for _, r := range raw {
		switch r.Category {
		case "Pilot":
			if n := strings.TrimSpace(r.Name); n != "" {
				names[strings.ToLower(n)] = true
			}
		case "Command":
			if r.Effect == nil {
				continue
			}
			idx := strings.Index(*r.Effect, models.PilotEffectMarker)
			if idx == -1 {
				continue
			}
			rest := (*r.Effect)[idx+len(models.PilotEffectMarker):]
			end := strings.Index(rest, "]")
			if end == -1 {
				continue
			}
			if n := strings.TrimSpace(rest[:end]); n != "" {
				names[strings.ToLower(n)] = true
			}
		}
	}
	return names
}

// fixLinkRequirement corrects two known export mistakes in link_requirement
// strings, using pilotNames (see collectPilotNames) to tell a pilot name
// apart from a genuine trait:
//
//   - a pilot name wrapped in "(...)" instead of "[...]", e.g. "(Char Aznable)"
//   - a bare pilot name with no delimiters at all, e.g. "M'Quve"
//
// Traits (including ones that happen to have no "Trait" suffix, e.g. "(Zeon)")
// are left untouched since they never match an entry in pilotNames.
func fixLinkRequirement(lr string, pilotNames map[string]bool) string {
	if lr == "" {
		return lr
	}
	if !strings.ContainsAny(lr, "[(") {
		if pilotNames[strings.ToLower(strings.TrimSpace(lr))] {
			return "[" + strings.TrimSpace(lr) + "]"
		}
		return lr
	}

	var out strings.Builder
	s := lr
	for {
		start := strings.Index(s, "(")
		if start == -1 {
			out.WriteString(s)
			break
		}
		end := strings.Index(s[start:], ")")
		if end == -1 {
			out.WriteString(s)
			break
		}
		seg := s[start+1 : start+end]
		out.WriteString(s[:start])
		if pilotNames[strings.ToLower(strings.TrimSpace(seg))] {
			out.WriteByte('[')
			out.WriteString(strings.TrimSpace(seg))
			out.WriteByte(']')
		} else {
			out.WriteByte('(')
			out.WriteString(seg)
			out.WriteByte(')')
		}
		s = s[start+end+1:]
	}
	return out.String()
}

// collectCanonicalTypes builds a lookup from a loose, delimiter-insensitive
// key to the canonical casing/spelling of a type as it appears elsewhere in
// the import — used to repair type entries that were exported malformed
// (see normalizeTypes).
func collectCanonicalTypes(raw []rawCard) map[string]string {
	canon := make(map[string]string)
	for _, r := range raw {
		for _, t := range r.Type {
			if strings.ContainsAny(t, "()") {
				continue // malformed entries are the ones we're trying to fix, not a source of truth
			}
			key := looseTypeKey(t)
			if _, ok := canon[key]; !ok {
				canon[key] = strings.TrimSpace(t)
			}
		}
	}
	return canon
}

// looseTypeKey normalizes a type string for comparison purposes only,
// ignoring case and hyphen-vs-space spelling differences.
func looseTypeKey(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), "-", " "))
}

// normalizeTypes repairs a card's raw type list. Some export rows pack
// multiple traits into a single string with stray parens instead of separate
// array entries, e.g. "(neo Zeon)(newtype)" instead of ["Neo Zeon", "Newtype"].
// Each "(...)" segment is split out and matched against canon to recover the
// correct casing; entries without parens are already well-formed and pass
// through unchanged.
func normalizeTypes(rawTypes []string, canon map[string]string) []string {
	var out []string
	for _, t := range rawTypes {
		for _, seg := range splitTypeSegments(t) {
			out = append(out, normalizeTypeSegment(seg, canon))
		}
	}
	return out
}

// splitTypeSegments extracts one or more "(...)" groups from a type string.
// A string with no parens is returned as a single-element slice unchanged.
func splitTypeSegments(raw string) []string {
	if !strings.Contains(raw, "(") {
		return []string{raw}
	}
	var segs []string
	s := raw
	for {
		start := strings.Index(s, "(")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], ")")
		if end == -1 {
			break
		}
		if seg := strings.TrimSpace(s[start+1 : start+end]); seg != "" {
			segs = append(segs, seg)
		}
		s = s[start+end+1:]
	}
	if len(segs) == 0 {
		return []string{raw}
	}
	return segs
}

// normalizeTypeSegment resolves a single type segment to its canonical form,
// preferring a match already seen elsewhere in the import and falling back to
// simple word capitalization for one that isn't attested anywhere clean.
func normalizeTypeSegment(seg string, canon map[string]string) string {
	seg = strings.TrimSpace(seg)
	if c, ok := canon[looseTypeKey(seg)]; ok {
		return c
	}
	words := strings.Fields(seg)
	for i, w := range words {
		r := []rune(w)
		if len(r) == 0 {
			continue
		}
		words[i] = strings.ToUpper(string(r[0])) + string(r[1:])
	}
	return strings.Join(words, " ")
}
