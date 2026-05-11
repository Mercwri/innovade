package deckbuilder

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Mercwri/innovade/internal/models"
)

type exportResultMsg struct {
	path string
	err  error
}

// formatDeckList produces the standard decklist format:
//
//	4 GD01-044 Kshatriya
//	2 ST08-002 Ξ Gundam
func formatDeckList(deck *models.Deck, nameFor func(string) string) string {
	var sb strings.Builder
	for _, e := range deck.Entries {
		name := nameFor(e.CardCode)
		if name == "" {
			name = e.CardCode
		}
		fmt.Fprintf(&sb, "%d %s %s\n", e.Quantity, e.CardCode, name)
	}
	return sb.String()
}

func exportDeckCmd(deck models.Deck, nameFor func(string) string) tea.Cmd {
	return func() tea.Msg {
		text := formatDeckList(&deck, nameFor)
		path := sanitizeFilename(deck.Name) + ".txt"
		err := os.WriteFile(path, []byte(text), 0644)
		return exportResultMsg{path: path, err: err}
	}
}

func sanitizeFilename(name string) string {
	var sb strings.Builder
	for _, r := range strings.TrimSpace(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			sb.WriteRune(unicode.ToLower(r))
		case r == ' ' || r == '-' || r == '_':
			sb.WriteRune('-')
		}
	}
	result := strings.Trim(sb.String(), "-")
	if result == "" {
		return "deck"
	}
	return result
}
