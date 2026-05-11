package models

import (
	"strings"
	"time"
)

type Category string

const (
	CategoryUnit      Category = "Unit"
	CategoryCommand   Category = "Command"
	CategoryPilot     Category = "Pilot"
	CategoryBase      Category = "Base"
	CategoryUnitToken Category = "Unit-token"
)

type Rarity string

const (
	RarityCommon    Rarity = "C"
	RarityUncommon  Rarity = "U"
	RarityRare      Rarity = "R"
	RarityLegendary Rarity = "LR"
)

type Color string

const (
	ColorBlue   Color = "Blue"
	ColorRed    Color = "Red"
	ColorGreen  Color = "Green"
	ColorWhite  Color = "White"
	ColorPurple Color = "Purple"
)

type Location string

const (
	LocationSpace Location = "Space"
	LocationEarth Location = "Earth"
)

type SetType string

const (
	SetTypeBooster SetType = "booster"
	SetTypeStarter SetType = "starter"
	SetTypeToken   SetType = "token"
)

type CardSet struct {
	Code    string // e.g. "GD01", "ST01", "T"
	Name    string // e.g. "Newtype Rising [GD01]"
	SetType SetType
}

func SetTypeFromCode(code string) SetType {
	if code == "T" {
		return SetTypeToken
	}
	if len(code) >= 2 && code[:2] == "ST" {
		return SetTypeStarter
	}
	return SetTypeBooster
}

type Card struct {
	// Identity
	ID       string // UUID from source data
	CardCode string // e.g. "GD01-001"
	SetCode  string // e.g. "GD01"
	SetName  string // e.g. "Newtype Rising [GD01]"
	Name     string
	Category Category
	Rarity   Rarity

	Colors []Color

	Cost  int // 0 for tokens
	Level int // 0 for tokens
	AP    int // 0 when not applicable
	HP    int // 0 when not applicable

	Types []string

	Locations []Location

	// Card text
	Effect          string // main effect text; empty for some tokens
	Burst           string // burst effect text; empty string = no burst
	LinkRequirement string // pilot name required for link, e.g. "[Amuro Ray]"

	// Images — index 0 is the default; additional entries are alternate arts
	ImagePaths       []string
	DefaultImagePath string

	CreatedAt time.Time
}

func (c *Card) IsUnit() bool {
	return c.Category == CategoryUnit || c.Category == CategoryUnitToken
}

func (c *Card) IsToken() bool {
	return c.Category == CategoryUnitToken
}

func (c *Card) HasBurst() bool {
	return c.Burst != ""
}

func (c *Card) HasLinkRequirement() bool {
	return c.LinkRequirement != ""
}

func (c *Card) HasLocation(loc Location) bool {
	for _, l := range c.Locations {
		if l == loc {
			return true
		}
	}
	return false
}

func (c *Card) HasType(t string) bool {
	for _, ct := range c.Types {
		if ct == t {
			return true
		}
	}
	return false
}

func (c *Card) PrimaryColor() Color {
	if len(c.Colors) == 0 {
		return ""
	}
	return c.Colors[0]
}

func (c *Card) AltArtCount() int {
	n := len(c.ImagePaths) - 1
	if n < 0 {
		return 0
	}
	return n
}

// ParseLinkRequirement extracts all link terms from a link requirement string.
//
// Two formats appear in the data, separated by " / " for OR logic:
//
//	[Amuro Ray]          → pilot name  "Amuro Ray"
//	(White Base Team) Trait → trait    "White Base Team"
//
// Both may appear together: "(Trinity) Trait / [Ali al-Saachez]"
func (c *Card) ParseLinkRequirement() []string {
	if c.LinkRequirement == "" {
		return nil
	}
	var terms []string

	// Pass 1: extract pilot names from [...]
	s := c.LinkRequirement
	for {
		start := strings.Index(s, "[")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "]")
		if end == -1 {
			break
		}
		if term := strings.TrimSpace(s[start+1 : start+end]); term != "" {
			terms = append(terms, term)
		}
		s = s[start+end+1:]
	}

	// Pass 2: extract trait names from (...)
	s = c.LinkRequirement
	for {
		start := strings.Index(s, "(")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], ")")
		if end == -1 {
			break
		}
		if term := strings.TrimSpace(s[start+1 : start+end]); term != "" {
			terms = append(terms, term)
		}
		s = s[start+end+1:]
	}

	return terms
}

// IsPilotEligible returns true for Pilot cards and Command cards that carry the Pilot trait.
func (c *Card) IsPilotEligible() bool {
	if c.Category == CategoryPilot {
		return true
	}
	if c.Category == CategoryCommand {
		for _, t := range c.Types {
			if strings.EqualFold(t, "pilot") {
				return true
			}
		}
	}
	return false
}
