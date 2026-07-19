package deckbuilder

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Mercwri/innovade/internal/models"
	"github.com/Mercwri/innovade/internal/ui/styles"
)

type exportResultMsg struct {
	path string
	note string // set for clipboard/link copies
	err  error
}

const exportOverlayW = 48

type exportAction int

const (
	exportActionNone exportAction = iota
	exportActionClose
	exportActionFile
	exportActionClipboard
	exportActionMSALink
	exportActionImage
)

var exportOptions = []struct {
	label  string
	action exportAction
}{
	{"Save to .txt file", exportActionFile},
	{"Copy decklist to clipboard", exportActionClipboard},
	{"Copy MSA link to clipboard", exportActionMSALink},
	{"Save as PNG image", exportActionImage},
}

// ExportPalette is the overlay for choosing an export format.
type ExportPalette struct{ cursor int }

func (p ExportPalette) Update(msg tea.Msg) (ExportPalette, exportAction) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, exportActionNone
	}
	switch {
	case key.Matches(km, importKeys.Close):
		return p, exportActionClose
	case key.Matches(km, importKeys.Up):
		if p.cursor > 0 {
			p.cursor--
		}
	case key.Matches(km, importKeys.Down):
		if p.cursor < len(exportOptions)-1 {
			p.cursor++
		}
	case key.Matches(km, importKeys.Select):
		return p, exportOptions[p.cursor].action
	}
	return p, exportActionNone
}

func (p ExportPalette) View() string {
	var sb strings.Builder
	itemW := exportOverlayW - 4

	divider := styles.StyleDetailDivider.Render(strings.Repeat("─", itemW))
	sb.WriteString(styles.StyleDetailTitle.Render("Export Deck"))
	sb.WriteString("\n")
	sb.WriteString(divider)
	sb.WriteString("\n")

	for i, opt := range exportOptions {
		label := lipgloss.NewStyle().Width(itemW).Render(opt.label)
		if i == p.cursor {
			sb.WriteString(styles.StylePaletteItemSelected.Render(label))
		} else {
			sb.WriteString(styles.StylePaletteItem.Render(label))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(divider)
	sb.WriteString("\n")
	sb.WriteString(styles.StylePaletteItem.Render(
		styles.StylePaletteHint.Render("↑↓/jk · enter select · esc close"),
	))

	return styles.StylePaletteOverlay.Width(exportOverlayW).Render(sb.String())
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

func formatMSALink(deck *models.Deck) string {
	var sb strings.Builder
	sb.WriteString("https://mobilesuitarena.com/?decklist=")
	for i, e := range deck.Entries {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, "%s:%d", e.CardCode, e.Quantity)
	}
	sb.WriteString("&type=gundam")
	return sb.String()
}

func copyToClipboardCmd(text, note string) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.WriteAll(text); err != nil {
			return exportResultMsg{err: fmt.Errorf("clipboard: %w", err)}
		}
		return exportResultMsg{note: note}
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
