package deckbuilder

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Mercwri/innovade/internal/models"
	"github.com/Mercwri/innovade/internal/store"
	"github.com/Mercwri/innovade/internal/ui/styles"
)

const importOverlayW = 48

var importKeys = struct {
	Close  key.Binding
	Select key.Binding
	Up     key.Binding
	Down   key.Binding
}{
	Close:  key.NewBinding(key.WithKeys("esc")),
	Select: key.NewBinding(key.WithKeys("enter")),
	Up:     key.NewBinding(key.WithKeys("up", "k")),
	Down:   key.NewBinding(key.WithKeys("down", "j")),
}

type importAction int

const (
	importActionNone   importAction = iota
	importActionClose
	importActionSelect
)

type importResultMsg struct {
	deckName string
	err      error
}

// ImportPalette lists .txt files in the current directory for selection.
type ImportPalette struct {
	files  []string
	cursor int
}

func NewImportPalette() ImportPalette {
	entries, err := os.ReadDir(".")
	if err != nil {
		return ImportPalette{}
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".txt") {
			files = append(files, e.Name())
		}
	}
	return ImportPalette{files: files}
}

func (p ImportPalette) Update(msg tea.Msg) (ImportPalette, importAction) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, importActionNone
	}
	switch {
	case key.Matches(km, importKeys.Close):
		return p, importActionClose
	case key.Matches(km, importKeys.Up):
		if p.cursor > 0 {
			p.cursor--
		}
	case key.Matches(km, importKeys.Down):
		if p.cursor < len(p.files)-1 {
			p.cursor++
		}
	case key.Matches(km, importKeys.Select):
		if len(p.files) > 0 {
			return p, importActionSelect
		}
	}
	return p, importActionNone
}

func (p ImportPalette) selectedFile() string {
	if len(p.files) == 0 || p.cursor >= len(p.files) {
		return ""
	}
	return p.files[p.cursor]
}

func (p ImportPalette) View() string {
	var sb strings.Builder
	itemW := importOverlayW - 4 // overlay has 2-char padding on each side

	divider := styles.StyleDetailDivider.Render(strings.Repeat("─", itemW))
	sb.WriteString(styles.StyleDetailTitle.Render("Import Deck"))
	sb.WriteString("\n")
	sb.WriteString(divider)
	sb.WriteString("\n")

	if len(p.files) == 0 {
		sb.WriteString(styles.StylePaletteItem.Render(
			styles.StyleDetailLabel.Width(itemW).Render("No .txt files found in current directory"),
		))
		sb.WriteString("\n")
	} else {
		for i, f := range p.files {
			label := lipgloss.NewStyle().Width(itemW).Render(truncate(f, itemW))
			if i == p.cursor {
				sb.WriteString(styles.StylePaletteItemSelected.Render(label))
			} else {
				sb.WriteString(styles.StylePaletteItem.Render(label))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString(divider)
	sb.WriteString("\n")
	sb.WriteString(styles.StylePaletteItem.Render(
		styles.StylePaletteHint.Render("↑↓/jk · enter import · esc close"),
	))

	return styles.StylePaletteOverlay.Width(importOverlayW).Render(sb.String())
}

// ── Commands ──────────────────────────────────────────────────────────────────

func importDeckCmd(s *store.Store, path string) tea.Cmd {
	return func() tea.Msg {
		deck, err := parseDeckFile(path)
		if err != nil {
			return importResultMsg{err: err}
		}
		if err := s.SaveDeck(deck); err != nil {
			return importResultMsg{err: fmt.Errorf("save: %w", err)}
		}
		return importResultMsg{deckName: deck.Name}
	}
}

// parseDeckFile parses the standard decklist format:
//
//	4 GD01-044 Kshatriya
//	2 ST08-002 Ξ Gundam
//
// The card name is optional — it is ignored in favour of the database value.
// Blank lines and lines starting with # or // are skipped.
func parseDeckFile(path string) (*models.Deck, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	deck := &models.Deck{}
	base := filepath.Base(path)
	deck.Name = strings.TrimSuffix(base, filepath.Ext(base))

	for lineNum, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: expected '<qty> <code> [name]', got %q", lineNum+1, line)
		}

		qty, err := strconv.Atoi(parts[0])
		if err != nil || qty <= 0 {
			return nil, fmt.Errorf("line %d: invalid quantity %q", lineNum+1, parts[0])
		}
		if qty > models.MaxCopiesPerCard {
			return nil, fmt.Errorf("line %d: %s — quantity %d exceeds max of %d", lineNum+1, parts[1], qty, models.MaxCopiesPerCard)
		}

		deck.Entries = append(deck.Entries, models.DeckEntry{
			CardCode: strings.ToUpper(parts[1]),
			Quantity: qty,
		})
	}

	if len(deck.Entries) == 0 {
		return nil, fmt.Errorf("no entries found in %q", filepath.Base(path))
	}
	return deck, nil
}
