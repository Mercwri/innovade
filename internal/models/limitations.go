package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Limitation string

const (
	LimitationBanned         Limitation = "Banned"
	LimitationRestricted     Limitation = "Restricted"
	LimitationCombination    Limitation = "Combination"
	LimitationClassification Limitation = "Classification"
	// Legacy spelling retained for compatibility with existing callers/tests.
	LimitationClassifcation Limitation = LimitationClassification
)

// Banned: cards that cannot be played.
// Restricted: cards that cannot be played at a 4 count and instead are played at a set quantity.
// Combination: cards that cannot be played together (eg pilot A and pilot B, or unit A and unit B).
// Classification: cards matching a specific signature type (eg level 2, 1 cost 2ap/hp) are treated like a Combination Match they cannot be in a deck with another matching type.

type LimitationStats struct {
	Level           int    `json:"level"`
	Cost            int    `json:"cost"`
	AP              int    `json:"ap"`
	HP              int    `json:"hp"`
	Effect          string `json:"effect"`
	Burst           string `json:"burst"`
	LinkRequirement string `json:"linkRequirement"`
}

type LimitationRule struct {
	Type     Limitation      `json:"type"`
	CardID   string          `json:"cardId,omitempty"`
	CardIDs  []string        `json:"cardIds,omitempty"`
	Quantity int             `json:"quantity,omitempty"`
	Stats    LimitationStats `json:"stats,omitempty"`
}

func LoadLimitationsFromPath(path string) ([]LimitationRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read limitations %s: %w", path, err)
	}

	var rules []LimitationRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("decode limitations %s: %w", path, err)
	}
	return rules, nil
}

func LoadLimitationsFromDataDir(dir string) ([]LimitationRule, error) {
	return LoadLimitationsFromPath(filepath.Join(dir, "bans.json"))
}

func (d *Deck) Validate(limitations []LimitationRule, cards map[string]Card) ValidationResult {
	return ValidateDeck(limitations, d, cards)
}

func ValidateDeck(limitations []LimitationRule, deck *Deck, cards map[string]Card) ValidationResult {
	result := ValidationResult{Valid: true}
	if deck == nil {
		return result
	}

	counts := make(map[string]int, len(deck.Entries))
	for _, entry := range deck.Entries {
		if entry.Quantity <= 0 {
			continue
		}
		counts[entry.CardCode] += entry.Quantity
	}

	for _, rule := range limitations {
		switch rule.Type {
		case LimitationBanned:
			if qty := counts[rule.CardID]; qty > 0 {
				result.Errors = append(result.Errors, ValidationError{
					Code:    "card_banned",
					Message: fmt.Sprintf("%s is banned", rule.CardID),
				})
			}
		case LimitationRestricted:
			maxQty := rule.Quantity
			if maxQty <= 0 {
				maxQty = 2
			}
			if qty := counts[rule.CardID]; qty > maxQty {
				result.Errors = append(result.Errors, ValidationError{
					Code:    "card_restricted",
					Message: fmt.Sprintf("%s exceeds the current restriction of %d copies", rule.CardID, maxQty),
				})
			}
		case LimitationCombination:
			if len(rule.CardIDs) < 2 {
				continue
			}
			first := rule.CardIDs[0]
			second := rule.CardIDs[1]
			if counts[first] > 0 && counts[second] > 0 {
				result.Errors = append(result.Errors, ValidationError{
					Code:    "combination_conflict",
					Message: fmt.Sprintf("%s and %s cannot be played together", first, second),
				})
			}
		default:
			if rule.Type != LimitationClassification && rule.Type != LimitationClassifcation {
				continue
			}
			matchingIDs := 0
			for code := range counts {
				card, ok := cards[code]
				if !ok {
					continue
				}
				if !cardMatchesStats(card, rule.Stats) {
					continue
				}
				matchingIDs++
				if matchingIDs > 1 {
					result.Errors = append(result.Errors, ValidationError{
						Code:    "classification_restricted",
						Message: fmt.Sprintf("%s and other matching cards cannot coexist in the same deck", code),
					})
					break
				}
			}
		}
	}

	for code, qty := range counts {
		if qty > MaxCopiesPerCard {
			result.Errors = append(result.Errors, ValidationError{
				Code:    "card_too_many_copies",
				Message: fmt.Sprintf("%s exceeds the max of %d copies", code, MaxCopiesPerCard),
			})
		}
	}

	if len(result.Errors) > 0 {
		result.Valid = false
	}
	return result
}

func cardMatchesStats(card Card, stats LimitationStats) bool {
	if stats.Level > 0 && card.Level != stats.Level {
		return false
	}
	if stats.Cost > 0 && card.Cost != stats.Cost {
		return false
	}
	if stats.AP > 0 && card.AP != stats.AP {
		return false
	}
	if stats.HP > 0 && card.HP != stats.HP {
		return false
	}
	if stats.Effect != "" && !strings.EqualFold(card.Effect, stats.Effect) {
		return false
	}
	if stats.Burst != "" && !strings.EqualFold(card.Burst, stats.Burst) {
		return false
	}
	if stats.LinkRequirement != "" && !strings.EqualFold(card.LinkRequirement, stats.LinkRequirement) {
		return false
	}
	return true
}
