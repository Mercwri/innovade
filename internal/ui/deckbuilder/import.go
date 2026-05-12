package deckbuilder

import (
	"fmt"
	"math/rand"
	"net/url"
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
	importActionNone importAction = iota
	importActionClose
	importActionSelect
	importActionURL
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
	// cursor 0 = "From URL..." entry; cursor 1..len(files) = file entries
	switch {
	case key.Matches(km, importKeys.Close):
		return p, importActionClose
	case key.Matches(km, importKeys.Up):
		if p.cursor > 0 {
			p.cursor--
		}
	case key.Matches(km, importKeys.Down):
		if p.cursor < len(p.files) {
			p.cursor++
		}
	case key.Matches(km, importKeys.Select):
		if p.cursor == 0 {
			return p, importActionURL
		}
		return p, importActionSelect
	}
	return p, importActionNone
}

func (p ImportPalette) selectedFile() string {
	if p.cursor == 0 || p.cursor > len(p.files) {
		return ""
	}
	return p.files[p.cursor-1]
}

func (p ImportPalette) View() string {
	var sb strings.Builder
	itemW := importOverlayW - 4 // overlay has 2-char padding on each side

	divider := styles.StyleDetailDivider.Render(strings.Repeat("─", itemW))
	sb.WriteString(styles.StyleDetailTitle.Render("Import Deck"))
	sb.WriteString("\n")
	sb.WriteString(divider)
	sb.WriteString("\n")

	// URL option is always first (cursor 0)
	urlLabel := lipgloss.NewStyle().Width(itemW).Render("↗ From URL...")
	if p.cursor == 0 {
		sb.WriteString(styles.StylePaletteItemSelected.Render(urlLabel))
	} else {
		sb.WriteString(styles.StylePaletteItem.Render(urlLabel))
	}
	sb.WriteString("\n")

	if len(p.files) == 0 {
		sb.WriteString(styles.StylePaletteItem.Render(
			styles.StyleDetailLabel.Width(itemW).Render("(no .txt files in current directory)"),
		))
		sb.WriteString("\n")
	} else {
		for i, f := range p.files {
			label := lipgloss.NewStyle().Width(itemW).Render(truncate(f, itemW))
			if p.cursor == i+1 {
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

// parseDeckUrl parses a deck list from a url
// https://deckbuilder.egmanevents.com/?deck=GD04-003:4,GD04-006:4,GD04-081:4,GD04-016:4,GD04-015:2,GD04-013:2,GD04-121:4,GD01-118:3,GD01-100:2,GD04-065:4,GD04-098:3,ST01-010:4,ST01-001:3,GD01-006:2,GD01-086:3,GD04-077:2&type=gundam
func parseDeckUrl(rawURL string) (*models.Deck, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	deckParam := u.Query().Get("deck")
	if deckParam == "" {
		return nil, fmt.Errorf("url missing 'deck' query parameter")
	}

	deck := &models.Deck{Name: fmt.Sprintf("Deck %04d", rand.Intn(10000))}

	for entry := range strings.SplitSeq(deckParam, ",") {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid entry %q: expected 'CODE:QTY'", entry)
		}
		qty, err := strconv.Atoi(parts[1])
		if err != nil || qty <= 0 {
			return nil, fmt.Errorf("invalid quantity in entry %q", entry)
		}
		if qty > models.MaxCopiesPerCard {
			return nil, fmt.Errorf("%s — quantity %d exceeds max of %d", parts[0], qty, models.MaxCopiesPerCard)
		}
		deck.Entries = append(deck.Entries, models.DeckEntry{
			CardCode: strings.ToUpper(parts[0]),
			Quantity: qty,
		})
	}

	if len(deck.Entries) == 0 {
		return nil, fmt.Errorf("no entries found in url")
	}
	return deck, nil
}

func importDeckUrlCmd(s *store.Store, rawURL string) tea.Cmd {
	return func() tea.Msg {
		deck, err := parseDeckUrl(rawURL)
		if err != nil {
			return importResultMsg{err: err}
		}
		if err := s.SaveDeck(deck); err != nil {
			return importResultMsg{err: fmt.Errorf("save: %w", err)}
		}
		return importResultMsg{deckName: deck.Name}
	}
}

func renderURLImportOverlay(input string, w, h int) string {
	const overlayW = 60
	itemW := overlayW - 4

	var sb strings.Builder
	divider := styles.StyleDetailDivider.Render(strings.Repeat("─", itemW))

	sb.WriteString(styles.StyleDetailTitle.Render("Import from URL"))
	sb.WriteString("\n")
	sb.WriteString(divider)
	sb.WriteString("\n")
	sb.WriteString(styles.StylePaletteItem.Render(
		lipgloss.NewStyle().Width(itemW).Render(truncate(input+"█", itemW)),
	))
	sb.WriteString("\n")
	sb.WriteString(divider)
	sb.WriteString("\n")
	sb.WriteString(styles.StylePaletteItem.Render(
		styles.StylePaletteHint.Render("paste url · enter import · esc cancel"),
	))

	overlay := styles.StylePaletteOverlay.Width(overlayW).Render(sb.String())
	y := max((h-lipgloss.Height(overlay))/4, 0)
	return lipgloss.Place(w, h,
		lipgloss.Center, lipgloss.Top,
		lipgloss.NewStyle().MarginTop(y).Render(overlay),
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceBackground(styles.BgBase),
	)
}
