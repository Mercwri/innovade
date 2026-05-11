package models

// CardFilter defines the criteria for querying cards from the store.
// Zero values mean "no filter" for that field — all filters are optional
// and combine with AND logic.
type CardFilter struct {
	// Text search against card name (case-insensitive substring)
	Name string

	// Enum filters — empty slice means "any"
	Categories []Category
	Rarities   []Rarity
	Colors     []Color
	Locations  []Location

	// Faction/archetype type — empty string means "any"
	Type string

	// Set filter — empty string means "any"
	SetCode string

	// Stat range filters — nil pointer means "no bound"
	MinCost  *int
	MaxCost  *int
	MinLevel *int
	MaxLevel *int
	MinAP    *int
	MaxAP    *int
	MinHP    *int
	MaxHP    *int

	// Mechanic filters
	HasBurst           *bool // nil = any, true = must have burst
	HasLinkRequirement *bool

	// Exclude tokens from results (common default for deck building)
	ExcludeTokens bool

	// FindPilots: include Pilot-category cards AND Command cards that have the
	// "Pilot" trait. Takes precedence over the Categories filter when set.
	FindPilots bool

	// PilotLinkNames: pilot names from a unit's "[Name]" link requirement terms.
	// Matches pilot-eligible cards whose name contains any of these.
	PilotLinkNames []string

	// PilotLinkTraits: trait names from a unit's "(Trait)" link requirement terms.
	// Matches pilot-eligible cards that have any of these as an exact card type.
	PilotLinkTraits []string

	// UnitLinkNames: a pilot's name. Matches Unit cards whose link_requirement
	// contains "[Name]" (bracket-bounded for precision).
	UnitLinkNames []string

	// UnitLinkTraits: a pilot's non-pilot traits. Matches Unit cards whose
	// link_requirement contains "(Trait)" (paren-bounded for precision).
	UnitLinkTraits []string

	// Description overrides the auto-generated status-bar summary for this filter.
	Description string
}

// SortField specifies which field to sort card results by.
type SortField string

const (
	SortByName     SortField = "name"
	SortByCost     SortField = "cost"
	SortByLevel    SortField = "level"
	SortByAP       SortField = "ap"
	SortByHP       SortField = "hp"
	SortBySetCode  SortField = "set_code"
	SortByCardCode SortField = "card_code"
	SortByRarity   SortField = "rarity"
)

// SortOrder specifies ascending or descending sort.
type SortOrder string

const (
	SortAsc  SortOrder = "ASC"
	SortDesc SortOrder = "DESC"
)

// CardQuery combines a filter with sort and pagination options.
type CardQuery struct {
	Filter CardFilter
	SortBy SortField
	Order  SortOrder
	Limit  int // 0 = no limit
	Offset int
}
