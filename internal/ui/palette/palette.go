package palette

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var StylePaletteInput = lipgloss.NewStyle().
	Padding(0, 1)

var StyleDetailDivider = lipgloss.NewStyle().
	Foreground(lipgloss.Color("240"))

var StylePaletteHint = lipgloss.NewStyle().
	Foreground(lipgloss.Color("241"))

var StylePaletteItem = lipgloss.NewStyle().
	Padding(0, 1)

var StylePaletteItemSelected = lipgloss.NewStyle().
	Padding(0, 1).
	Background(lipgloss.Color("7")).
	Foreground(lipgloss.Color("0"))

var StylePaletteOverlay = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("62")).
	Padding(1)

// PaletteAction represents a resolved command from the palette.
type PaletteAction int

const (
	ActionNone PaletteAction = iota
	ActionQuit
	ActionOpenLibrary
	ActionOpenDeckBuilder
	ActionOpenDeckLibrary
	ActionOpenAnalysis
	ActionFilterLibrary
	ActionSearchLibrary
	ActionClearFilters
	ActionImportCards
)

type View int

const (
	ViewGlobal View = iota - 1
	ViewLibrary
)

type paletteEntry struct {
	label   string
	hint    string
	action  PaletteAction
	context View // -1 means global
}

var Palette = struct {
	Close  key.Binding
	Select key.Binding
	Up     key.Binding
	Down   key.Binding
}{
	Close: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close palette"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select command"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("up/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("down/j", "move down"),
	),
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
	{label: "Filter by Color", hint: "narrow by card color", action: ActionFilterLibrary},
	{label: "Filter by Category", hint: "unit, command, pilot...", action: ActionFilterLibrary},
	{label: "Filter by Set", hint: "narrow by set code", action: ActionFilterLibrary},
	{label: "Search by Name", hint: "find cards by name", action: ActionSearchLibrary},
	{label: "Clear Filters", hint: "reset all active filters", action: ActionClearFilters},
}

// PaletteModel is the command palette overlay.
type PaletteModel struct {
	input    textinput.Model
	entries  []paletteEntry
	filtered []paletteEntry
	cursor   int
	context  View
}

func NewPaletteModel(ctx View) PaletteModel {
	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = 46

	entries := append(globalEntries, contextEntries(ctx)...)

	m := PaletteModel{
		input:   ti,
		entries: entries,
		context: ctx,
	}
	m.filtered = entries
	return m
}

func contextEntries(v View) []paletteEntry {
	switch v {
	case ViewLibrary:
		return libraryEntries
	}
	return nil
}

// Update returns the updated model, a resolved action (if any), and a Cmd.
func (m PaletteModel) Update(msg tea.Msg) (PaletteModel, PaletteAction, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Palette.Close):
			return m, ActionNone, nil

		case key.Matches(msg, Palette.Select):
			if len(m.filtered) > 0 {
				return m, m.filtered[m.cursor].action, nil
			}
			return m, ActionNone, nil

		case key.Matches(msg, Palette.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, ActionNone, nil

		case key.Matches(msg, Palette.Down):
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, ActionNone, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.filtered = m.filter(m.input.Value())
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	return m, ActionNone, cmd
}

func (m PaletteModel) filter(query string) []paletteEntry {
	if query == "" {
		return m.entries
	}
	q := strings.ToLower(query)
	var out []paletteEntry
	for _, e := range m.entries {
		if strings.Contains(strings.ToLower(e.label), q) ||
			strings.Contains(strings.ToLower(e.hint), q) {
			out = append(out, e)
		}
	}
	return out
}

func (m PaletteModel) View() string {
	var sb strings.Builder

	sb.WriteString(StylePaletteInput.Render(m.input.View()))
	sb.WriteString("\n")
	sb.WriteString(StyleDetailDivider.Render(strings.Repeat("─", 48)))
	sb.WriteString("\n")

	maxVisible := 8
	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	if len(m.filtered) == 0 {
		sb.WriteString(StylePaletteHint.Render("  no matching commands"))
		sb.WriteString("\n")
	}

	for i := start; i < end; i++ {
		e := m.filtered[i]
		label := lipgloss.NewStyle().Width(22).Render(e.label)
		hint := StylePaletteHint.Width(24).Render(e.hint)
		row := label + hint

		if i == m.cursor {
			sb.WriteString(StylePaletteItemSelected.Render(row))
		} else {
			sb.WriteString(StylePaletteItem.Render(row))
		}
		sb.WriteString("\n")
	}

	if len(m.filtered) > maxVisible {
		sb.WriteString(StylePaletteHint.Render("  … more results"))
		sb.WriteString("\n")
	}

	return StylePaletteOverlay.
		Width(50).
		Render(sb.String())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
