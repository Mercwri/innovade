package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/Mercwri/innovade/internal/models"
)

// GetCard retrieves a single card by its card code (e.g. "GD01-001").
// Returns sql.ErrNoRows if not found.
func (s *Store) GetCard(cardCode string) (*models.Card, error) {
	row := s.db.QueryRow(`
		SELECT id, card_code, set_code, name, category, rarity, created_at,
		       cost, level, ap, hp, effect, burst, link_requirement, default_image_path
		FROM cards WHERE card_code = ?`, cardCode)

	card, err := scanCard(row)
	if err != nil {
		return nil, err
	}
	if err := s.hydrateCard(card); err != nil {
		return nil, err
	}
	return card, nil
}

// QueryCards returns cards matching the given query, with sort and pagination.
func (s *Store) QueryCards(q models.CardQuery) ([]models.Card, error) {
	sb := &strings.Builder{}
	args := []any{}

	sb.WriteString(`
		SELECT DISTINCT c.id, c.card_code, c.set_code, c.name, c.category, c.rarity,
		       c.created_at, c.cost, c.level, c.ap, c.hp,
		       c.effect, c.burst, c.link_requirement, c.default_image_path
		FROM cards c
	`)

	// JOIN only when filtering on child tables
	f := q.Filter
	if len(f.Colors) > 0 {
		sb.WriteString(" JOIN card_colors cc ON cc.card_code = c.card_code")
	}
	if f.Type != "" {
		sb.WriteString(" JOIN card_types ct ON ct.card_code = c.card_code")
	}
	if len(f.Locations) > 0 {
		sb.WriteString(" JOIN card_locations cl ON cl.card_code = c.card_code")
	}

	conditions := []string{}

	if f.Name != "" {
		conditions = append(conditions, "c.name LIKE ?")
		args = append(args, "%"+f.Name+"%")
	}
	if f.FindPilots {
		// Pilot category OR Command cards with the garbled 【Pilot】[Name] marker in their effect.
		conditions = append(conditions, "(c.category = 'Pilot' OR (c.category = 'Command' AND c.effect LIKE ?))")
		args = append(args, "%"+models.PilotEffectMarker+"%")
	} else if len(f.Categories) > 0 {
		conditions = append(conditions, "c.category IN ("+placeholders(len(f.Categories))+")")
		for _, cat := range f.Categories {
			args = append(args, string(cat))
		}
	}
	if len(f.PilotLinkNames) > 0 || len(f.PilotLinkTraits) > 0 {
		// Match pilots by name (Pilot cards) OR by pilot-name in effect (Command-Pilot cards).
		var termConds []string
		for _, name := range f.PilotLinkNames {
			termConds = append(termConds, "(c.name LIKE ? OR c.effect LIKE ?)")
			args = append(args, "%"+name+"%", "%["+name+"]%")
		}
		for _, trait := range f.PilotLinkTraits {
			termConds = append(termConds, "EXISTS (SELECT 1 FROM card_types ct_lt WHERE ct_lt.card_code = c.card_code AND LOWER(ct_lt.type) = LOWER(?))")
			args = append(args, trait)
		}
		conditions = append(conditions, "("+strings.Join(termConds, " OR ")+")")
	}
	if len(f.UnitLinkNames) > 0 || len(f.UnitLinkTraits) > 0 {
		// Use delimiter-bounded patterns to avoid false matches (e.g. Newtype vs Cyber-Newtype).
		var termConds []string
		for _, name := range f.UnitLinkNames {
			termConds = append(termConds, "c.link_requirement LIKE ?")
			args = append(args, "%["+name+"]%")
		}
		for _, trait := range f.UnitLinkTraits {
			termConds = append(termConds, "c.link_requirement LIKE ?")
			args = append(args, "%("+trait+")%")
		}
		conditions = append(conditions, "("+strings.Join(termConds, " OR ")+")")
	}
	if f.ExcludeTokens {
		conditions = append(conditions, "c.category != 'Unit-token'")
	}
	if len(f.Rarities) > 0 {
		conditions = append(conditions, "c.rarity IN ("+placeholders(len(f.Rarities))+")")
		for _, r := range f.Rarities {
			args = append(args, string(r))
		}
	}
	if len(f.Colors) > 0 {
		conditions = append(conditions, "cc.color IN ("+placeholders(len(f.Colors))+")")
		for _, col := range f.Colors {
			args = append(args, string(col))
		}
	}
	if f.Type != "" {
		conditions = append(conditions, "ct.type = ?")
		args = append(args, f.Type)
	}
	if len(f.Locations) > 0 {
		conditions = append(conditions, "cl.location IN ("+placeholders(len(f.Locations))+")")
		for _, loc := range f.Locations {
			args = append(args, string(loc))
		}
	}
	if len(f.SetCodes) > 0 {
		conditions = append(conditions, "c.set_code IN ("+placeholders(len(f.SetCodes))+")")
		for _, sc := range f.SetCodes {
			args = append(args, sc)
		}
	}
	if f.MinCost != nil {
		conditions = append(conditions, "c.cost >= ?")
		args = append(args, *f.MinCost)
	}
	if f.MaxCost != nil {
		conditions = append(conditions, "c.cost <= ?")
		args = append(args, *f.MaxCost)
	}
	if f.MinLevel != nil {
		conditions = append(conditions, "c.level >= ?")
		args = append(args, *f.MinLevel)
	}
	if f.MaxLevel != nil {
		conditions = append(conditions, "c.level <= ?")
		args = append(args, *f.MaxLevel)
	}
	if f.MinAP != nil {
		conditions = append(conditions, "c.ap >= ?")
		args = append(args, *f.MinAP)
	}
	if f.MaxAP != nil {
		conditions = append(conditions, "c.ap <= ?")
		args = append(args, *f.MaxAP)
	}
	if f.MinHP != nil {
		conditions = append(conditions, "c.hp >= ?")
		args = append(args, *f.MinHP)
	}
	if f.MaxHP != nil {
		conditions = append(conditions, "c.hp <= ?")
		args = append(args, *f.MaxHP)
	}
	if f.HasBurst != nil {
		if *f.HasBurst {
			conditions = append(conditions, "c.burst != ''")
		} else {
			conditions = append(conditions, "c.burst = ''")
		}
	}
	if f.HasLinkRequirement != nil {
		if *f.HasLinkRequirement {
			conditions = append(conditions, "c.link_requirement != ''")
		} else {
			conditions = append(conditions, "c.link_requirement = ''")
		}
	}

	if len(conditions) > 0 {
		sb.WriteString(" WHERE " + strings.Join(conditions, " AND "))
	}

	// Sort
	sortCol := "c.card_code" // safe default
	validSorts := map[models.SortField]string{
		models.SortByName:     "c.name COLLATE NOCASE",
		models.SortByCost:     "c.cost",
		models.SortByLevel:    "c.level",
		models.SortByAP:       "c.ap",
		models.SortByHP:       "c.hp",
		models.SortBySetCode:  "c.set_code",
		models.SortByCardCode: "c.card_code",
		models.SortByRarity:   "c.rarity",
	}
	if col, ok := validSorts[q.SortBy]; ok {
		sortCol = col
	}
	order := "ASC"
	if q.Order == models.SortDesc {
		order = "DESC"
	}
	sb.WriteString(fmt.Sprintf(" ORDER BY %s %s", sortCol, order))

	if q.Limit > 0 {
		sb.WriteString(" LIMIT ?")
		args = append(args, q.Limit)
		if q.Offset > 0 {
			sb.WriteString(" OFFSET ?")
			args = append(args, q.Offset)
		}
	}

	rows, err := s.db.Query(sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query cards: %w", err)
	}
	defer rows.Close()

	var cards []models.Card
	for rows.Next() {
		card, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, *card)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Hydrate child fields (colors, types, locations, images) for all cards
	// in a single pass per table to avoid N+1 queries.
	if err := s.hydrateCards(cards); err != nil {
		return nil, err
	}

	return cards, nil
}

// CardStat holds the fields needed for deck analysis (curve grid, etc.).
type CardStat struct {
	Name     string
	Category models.Category
	Level    int
	Colors   []models.Color
}

// GetCardStatsByCodes returns a CardStat for each code provided.
func (s *Store) GetCardStatsByCodes(codes []string) (map[string]CardStat, error) {
	if len(codes) == 0 {
		return map[string]CardStat{}, nil
	}
	ph := placeholders(len(codes))
	args := make([]any, len(codes))
	for i, c := range codes {
		args[i] = c
	}

	rows, err := s.db.Query(
		"SELECT card_code, name, category, level FROM cards WHERE card_code IN ("+ph+")",
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("get card stats: %w", err)
	}
	result := make(map[string]CardStat, len(codes))
	for rows.Next() {
		var cs CardStat
		var code string
		if err := rows.Scan(&code, &cs.Name, (*string)(&cs.Category), &cs.Level); err != nil {
			rows.Close()
			return nil, err
		}
		result[code] = cs
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	colorRows, err := s.db.Query(
		"SELECT card_code, color FROM card_colors WHERE card_code IN ("+ph+")",
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("get card stat colors: %w", err)
	}
	defer colorRows.Close()
	for colorRows.Next() {
		var code, color string
		if err := colorRows.Scan(&code, &color); err != nil {
			return nil, err
		}
		if cs, ok := result[code]; ok {
			cs.Colors = append(cs.Colors, models.Color(color))
			result[code] = cs
		}
	}
	return result, colorRows.Err()
}

// GetCardsByCodeMap returns the full Card record for each provided code.
// Colours, types, locations, and images are hydrated in bulk.
func (s *Store) GetCardsByCodeMap(codes []string) (map[string]models.Card, error) {
	if len(codes) == 0 {
		return map[string]models.Card{}, nil
	}
	ph := placeholders(len(codes))
	args := make([]any, len(codes))
	for i, c := range codes {
		args[i] = c
	}

	rows, err := s.db.Query(`
		SELECT id, card_code, set_code, name, category, rarity, created_at,
		       cost, level, ap, hp, effect, burst, link_requirement, default_image_path
		FROM cards WHERE card_code IN (`+ph+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("get cards by code: %w", err)
	}

	var cardSlice []models.Card
	for rows.Next() {
		card, err := scanCard(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		cardSlice = append(cardSlice, *card)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := s.hydrateCards(cardSlice); err != nil {
		return nil, err
	}

	result := make(map[string]models.Card, len(cardSlice))
	for _, c := range cardSlice {
		result[c.CardCode] = c
	}
	return result, nil
}

// GetCardNamesByCodes returns a map of card code → name for the given codes.
func (s *Store) GetCardNamesByCodes(codes []string) (map[string]string, error) {
	if len(codes) == 0 {
		return map[string]string{}, nil
	}
	ph := placeholders(len(codes))
	args := make([]any, len(codes))
	for i, c := range codes {
		args[i] = c
	}
	rows, err := s.db.Query("SELECT card_code, name FROM cards WHERE card_code IN ("+ph+")", args...)
	if err != nil {
		return nil, fmt.Errorf("get card names: %w", err)
	}
	defer rows.Close()
	result := make(map[string]string, len(codes))
	for rows.Next() {
		var code, name string
		if err := rows.Scan(&code, &name); err != nil {
			return nil, err
		}
		result[code] = name
	}
	return result, rows.Err()
}

// GetAllSets returns all known card sets ordered by code.
func (s *Store) GetAllSets() ([]models.CardSet, error) {
	rows, err := s.db.Query(`SELECT code, name, set_type FROM card_sets ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sets []models.CardSet
	for rows.Next() {
		var cs models.CardSet
		var st string
		if err := rows.Scan(&cs.Code, &cs.Name, &st); err != nil {
			return nil, err
		}
		cs.SetType = models.SetType(st)
		sets = append(sets, cs)
	}
	return sets, rows.Err()
}

// CountCards returns the number of cards matching the filter (ignores pagination).
func (s *Store) CountCards(f models.CardFilter) (int, error) {
	q := models.CardQuery{Filter: f}
	// Re-use QueryCards but only ask for count — build minimal query.
	cards, err := s.QueryCards(q)
	if err != nil {
		return 0, err
	}
	return len(cards), nil
}

// --- scanning helpers ---

type scanner interface {
	Scan(dest ...any) error
}

func scanCard(row scanner) (*models.Card, error) {
	var c models.Card
	var createdAt string
	err := row.Scan(
		&c.ID, &c.CardCode, &c.SetCode, &c.Name,
		(*string)(&c.Category), (*string)(&c.Rarity),
		&createdAt,
		&c.Cost, &c.Level, &c.AP, &c.HP,
		&c.Effect, &c.Burst, &c.LinkRequirement,
		&c.DefaultImagePath,
	)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("scan card: %w", err)
	}
	return &c, nil
}

// hydrateCard populates child fields for a single card.
func (s *Store) hydrateCard(c *models.Card) error {
	cards := []models.Card{*c}
	if err := s.hydrateCards(cards); err != nil {
		return err
	}
	*c = cards[0]
	return nil
}

// hydrateCards bulk-loads colors, types, locations, and images for a slice of
// cards using one query per child table, avoiding N+1 queries.
func (s *Store) hydrateCards(cards []models.Card) error {
	if len(cards) == 0 {
		return nil
	}

	codes := make([]any, len(cards))
	index := make(map[string]int, len(cards))
	for i, c := range cards {
		codes[i] = c.CardCode
		index[c.CardCode] = i
	}
	ph := placeholders(len(codes))

	// Colors
	rows, err := s.db.Query("SELECT card_code, color FROM card_colors WHERE card_code IN ("+ph+")", codes...)
	if err != nil {
		return fmt.Errorf("hydrate colors: %w", err)
	}
	for rows.Next() {
		var code, color string
		if err := rows.Scan(&code, &color); err != nil {
			rows.Close()
			return err
		}
		i := index[code]
		cards[i].Colors = append(cards[i].Colors, models.Color(color))
	}
	rows.Close()

	// Types
	rows, err = s.db.Query("SELECT card_code, type FROM card_types WHERE card_code IN ("+ph+") ORDER BY sort_order", codes...)
	if err != nil {
		return fmt.Errorf("hydrate types: %w", err)
	}
	for rows.Next() {
		var code, t string
		if err := rows.Scan(&code, &t); err != nil {
			rows.Close()
			return err
		}
		i := index[code]
		cards[i].Types = append(cards[i].Types, t)
	}
	rows.Close()

	// Locations
	rows, err = s.db.Query("SELECT card_code, location FROM card_locations WHERE card_code IN ("+ph+")", codes...)
	if err != nil {
		return fmt.Errorf("hydrate locations: %w", err)
	}
	for rows.Next() {
		var code, loc string
		if err := rows.Scan(&code, &loc); err != nil {
			rows.Close()
			return err
		}
		i := index[code]
		cards[i].Locations = append(cards[i].Locations, models.Location(loc))
	}
	rows.Close()

	// Images
	rows, err = s.db.Query("SELECT card_code, image_path FROM card_images WHERE card_code IN ("+ph+") ORDER BY sort_order", codes...)
	if err != nil {
		return fmt.Errorf("hydrate images: %w", err)
	}
	for rows.Next() {
		var code, path string
		if err := rows.Scan(&code, &path); err != nil {
			rows.Close()
			return err
		}
		i := index[code]
		cards[i].ImagePaths = append(cards[i].ImagePaths, path)
	}
	rows.Close()

	return nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}
