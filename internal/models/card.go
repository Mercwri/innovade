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

// pilotEffectMarker is the byte-exact suffix that follows the word "Pilot" in the
// 【Pilot】[Name] effect marker on Command-Pilot cards. The source JSON had its
// Japanese lenticular brackets double-encoded (UTF-8 bytes read as Windows-1252
// then re-encoded), producing this specific three-codepoint sequence before "[".
const PilotEffectMarker = "Pilot】["

// ExtractPilotName parses the pilot name from the 【Pilot】[Name] marker embedded
// in a Command-Pilot card's effect text. Returns "" if no marker is found.
func (c *Card) ExtractPilotName() string {
	idx := strings.Index(c.Effect, PilotEffectMarker)
	if idx == -1 {
		return ""
	}
	rest := c.Effect[idx+len(PilotEffectMarker):]
	end := strings.Index(rest, "]")
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}

// AliasMarker precedes a pilot name in the "This card's name is also treated
// as [Name]" clause some Pilot cards use to substitute for another named
// pilot for linking purposes. It can appear in either Effect or Burst text.
const AliasMarker = "treated as ["

// AliasNames returns every pilot name this card's own name is also treated
// as, per the AliasMarker clause in its Effect and/or Burst text.
func (c *Card) AliasNames() []string {
	var names []string
	for _, text := range [2]string{c.Effect, c.Burst} {
		pos := 0
		for {
			rel := strings.Index(text[pos:], AliasMarker)
			if rel == -1 {
				break
			}
			start := pos + rel + len(AliasMarker)
			end := strings.Index(text[start:], "]")
			if end == -1 {
				break
			}
			if name := strings.TrimSpace(text[start : start+end]); name != "" {
				names = append(names, name)
			}
			pos = start + end + 1
		}
	}
	return names
}

// ParseLinkTerms splits the link requirement into pilot names (from "[Name]") and
// trait names (from "(Trait)"), returning them separately for precise SQL matching.
//
// Bracketed names are cut out of the string before the trait pass runs, so a
// name containing its own parens (e.g. "[Amate Yuzuriha (Machu)]") doesn't leak
// a bogus trait term.
func (c *Card) ParseLinkTerms() (names []string, traits []string) {
	if c.LinkRequirement == "" {
		return nil, nil
	}
	var traitScan strings.Builder
	s := c.LinkRequirement
	for {
		start := strings.Index(s, "[")
		if start == -1 {
			traitScan.WriteString(s)
			break
		}
		end := strings.Index(s[start:], "]")
		if end == -1 {
			traitScan.WriteString(s)
			break
		}
		if term := strings.TrimSpace(s[start+1 : start+end]); term != "" {
			names = append(names, term)
		}
		traitScan.WriteString(s[:start])
		s = s[start+end+1:]
	}
	s = traitScan.String()
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
			traits = append(traits, term)
		}
		s = s[start+end+1:]
	}
	return names, traits
}

// IsPilotEligible returns true for Pilot cards and Command cards that embed a
// 【Pilot】[Name] marker in their effect text.
func (c *Card) IsPilotEligible() bool {
	if c.Category == CategoryPilot {
		return true
	}
	if c.Category == CategoryCommand {
		return c.ExtractPilotName() != ""
	}
	return false
}
