package models

import (
	"strings"
	"testing"
)

func TestValidateDeckEnforcesLimitations(t *testing.T) {
	limitations := []LimitationRule{
		{Type: LimitationBanned, CardID: "GD01-020"},
		{Type: LimitationRestricted, CardID: "ST02-016", Quantity: 2},
		{Type: LimitationCombination, CardIDs: []string{"GD01-008", "GD05-015"}},
		{Type: LimitationClassifcation, Stats: LimitationStats{Level: 2, Cost: 1, AP: 2, HP: 2, Effect: "", Burst: "", LinkRequirement: ""}},
	}

	cards := map[string]Card{
		"GD01-020": {CardCode: "GD01-020"},
		"ST02-016": {CardCode: "ST02-016"},
		"GD01-008": {CardCode: "GD01-008"},
		"GD05-015": {CardCode: "GD05-015"},
		"GD01-021": {CardCode: "GD01-021", Level: 2, Cost: 1, AP: 2, HP: 2},
		"GD01-022": {CardCode: "GD01-022", Level: 2, Cost: 1, AP: 2, HP: 2},
	}

	deck := &Deck{Entries: []DeckEntry{
		{CardCode: "GD01-020", Quantity: 1},
		{CardCode: "ST02-016", Quantity: 3},
		{CardCode: "GD01-008", Quantity: 1},
		{CardCode: "GD05-015", Quantity: 1},
		{CardCode: "GD01-021", Quantity: 1},
		{CardCode: "GD01-022", Quantity: 1},
	}}

	result := ValidateDeck(limitations, deck, cards)
	if result.Valid {
		t.Fatalf("expected deck to be invalid, got valid result")
	}

	codes := make([]string, 0, len(result.Errors))
	for _, err := range result.Errors {
		codes = append(codes, err.Code)
	}

	expected := []string{"card_banned", "card_restricted", "combination_conflict", "classification_restricted"}
	for _, want := range expected {
		found := false
		for _, code := range codes {
			if code == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected validation error %q, got %s", want, strings.Join(codes, ", "))
		}
	}
}

func TestValidateDeckAllowsSingleMatchingClassificationCardID(t *testing.T) {
	limitations := []LimitationRule{{Type: LimitationClassifcation, Stats: LimitationStats{Level: 2, Cost: 1, AP: 2, HP: 2}}}
	cards := map[string]Card{
		"GD01-021": {CardCode: "GD01-021", Level: 2, Cost: 1, AP: 2, HP: 2},
	}
	deck := &Deck{Entries: []DeckEntry{{CardCode: "GD01-021", Quantity: 2}}}

	result := ValidateDeck(limitations, deck, cards)
	if !result.Valid {
		t.Fatalf("expected a single matching card ID to be allowed, got errors: %v", result.Errors)
	}
}
