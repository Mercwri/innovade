package palette

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Mercwri/innovade/internal/ui/styles"
)

// PaletteAction represents a resolved command from the palette.
type PaletteAction int

const (
	ActionNone PaletteAction = iota
	ActionClose
	ActionQuit
	ActionOpenLibrary
	ActionOpenDeckBuilder
	ActionOpenDeckLibrary
	ActionOpenAnalysis
	ActionFilterLibrary
	ActionImportCards
	// Kept for compatibility but no longer dispatched.
	ActionSearchLibrary
	ActionClearFilters
)

type View int

const (
	ViewGlobal View = iota - 1
	ViewLibrary
)

// Palette exposes key bindings so the parent model can forward events.
var Palette = struct {
	Close  key.Binding
	Select key.Binding
	Up     key.Binding
	Down   key.Binding
}{
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
	Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
}

type paletteEntry struct {
	label  string
	hint   string
	action PaletteAction
}

var globalEntries = []paletteEntry{
	{label: "Card Library", hint: "browse and search cards", action: ActionOpenLibrary},
	{label: "Deck Builder", hint: "build a new deck", action: ActionOpenDeckBuilder},
	{label: "Deck Library", hint: "manage saved decks", action: ActionOpenDeckLibrary},
	{label: "Analysis", hint: "analyse a deck", action: ActionOpenAnalysis},
	{label: "Import Cards", hint: "load cards from JSON", action: ActionImportCards},
	{label: "Quit", hint: "exit innovade", action: ActionQuit},
}

var libraryEntries = []paletteEntry{
	{label: "Filter Cards", hint: "search & filter", action: ActionFilterLibrary},
}

// column widths; label+hint must stay within overlayW-2 (item padding).
const (
	overlayW = 48
	labelW   = 20
	hintW    = 26
)

// PaletteModel is the command palette overlay.
type PaletteModel struct {
	entries    []paletteEntry
	numGlobal  int
	cursor     int
}

func NewPaletteModel(ctx View) PaletteModel {
	ctx_entries := contextEntries(ctx)
	entries := make([]paletteEntry, 0, len(globalEntries)+len(ctx_entries))
	entries = append(entries, globalEntries...)
	entries = append(entries, ctx_entries...)

	return PaletteModel{
		entries:   entries,
		numGlobal: len(globalEntries),
	}
}

func contextEntries(v View) []paletteEntry {
	switch v {
	case ViewLibrary:
		return libraryEntries
	}
	return nil
}

// Update returns the updated model, a resolved action, and a Cmd.
func (m PaletteModel) Update(msg tea.Msg) (PaletteModel, PaletteAction, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, ActionNone, nil
	}
	switch {
	case key.Matches(km, Palette.Close):
		return m, ActionClose, nil

	case key.Matches(km, Palette.Select):
		if len(m.entries) > 0 {
			return m, m.entries[m.cursor].action, nil
		}

	case key.Matches(km, Palette.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(km, Palette.Down):
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}
	}
	return m, ActionNone, nil
}

func (m PaletteModel) View() string {
	var sb strings.Builder

	divider := styles.StyleDetailDivider.Render(strings.Repeat("─", labelW+hintW))

	sb.WriteString(divider)
	sb.WriteString("\n")

	for i, e := range m.entries {
		if i == m.numGlobal && i > 0 {
			sb.WriteString(divider)
			sb.WriteString("\n")
		}

		label := lipgloss.NewStyle().Width(labelW).Render(e.label)
		hint := styles.StylePaletteHint.Width(hintW).Render(e.hint)
		row := label + hint

		if i == m.cursor {
			sb.WriteString(styles.StylePaletteItemSelected.Render(row))
		} else {
			sb.WriteString(styles.StylePaletteItem.Render(row))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(divider)
	sb.WriteString("\n")
	sb.WriteString(styles.StylePaletteItem.Render(
		styles.StylePaletteHint.Render("↑↓/jk · enter select · esc close"),
	))

	return styles.StylePaletteOverlay.Width(overlayW).Render(sb.String())
}
