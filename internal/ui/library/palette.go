package library

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Mercwri/innovade/internal/models"
	"github.com/Mercwri/innovade/internal/ui/styles"
)

// FilterAction is returned from FilterPalette.Update to signal the parent model.
type FilterAction int

const (
	FilterActionNone    FilterAction = iota
	FilterActionClose   // Esc in phase 1 — close palette, no change
	FilterActionApplied // Filter dispatched via cmd — close palette
	FilterActionToggle  // Flip filtersActive — close palette and reload
)

type filterPhase int

const (
	phaseTypeSelect filterPhase = iota
	phaseColorSelect
	phaseCategorySelect
)

// ── Phase 1 entries ───────────────────────────────────────────────────────────

type filterTypeEntry struct {
	label    string
	hint     string
	phase    filterPhase
	clearAll bool
	isToggle bool
}

// Index 0 is the meta toggle (label rendered dynamically).
var filterTypeEntries = []filterTypeEntry{
	{isToggle: true},
	{label: "Filter by Color", hint: "narrow by card color", phase: phaseColorSelect},
	{label: "Filter by Category", hint: "unit, command, pilot...", phase: phaseCategorySelect},
	{label: "Clear All Filters", hint: "reset active filters", clearAll: true},
}

// ── Phase 2: color list ───────────────────────────────────────────────────────

type colorEntry struct {
	color models.Color
	label string
}

var colorEntries = []colorEntry{
	{label: "Any  (clear color filter)", color: ""},
	{models.ColorBlue, "Blue"},
	{models.ColorRed, "Red"},
	{models.ColorGreen, "Green"},
	{models.ColorWhite, "White"},
	{models.ColorPurple, "Purple"},
}

// ── Phase 2: category list ────────────────────────────────────────────────────

type categoryEntry struct {
	category models.Category
	label    string
}

var categoryEntries = []categoryEntry{
	{label: "Any  (clear category filter)", category: ""},
	{models.CategoryUnit, "Unit"},
	{models.CategoryCommand, "Command"},
	{models.CategoryPilot, "Pilot"},
	{models.CategoryBase, "Base"},
	{models.CategoryUnitToken, "Unit Token"},
}

// ── Keys ──────────────────────────────────────────────────────────────────────

var filterKeys = struct {
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

// column widths; labelW+hintW must equal overlayW-2 (item padding eats 2).
const (
	fOverlayW = 46
	fLabelW   = 20
	fHintW    = 24
)

// ── FilterPalette ─────────────────────────────────────────────────────────────

// FilterPalette is a two-phase state machine: pick a filter type, then a value.
type FilterPalette struct {
	phase         filterPhase
	cursor        int
	current       models.CardFilter
	filtersActive bool
}

func NewFilterPalette(current models.CardFilter, active bool) FilterPalette {
	return FilterPalette{
		phase:         phaseTypeSelect,
		current:       current,
		filtersActive: active,
	}
}

func (m FilterPalette) Update(msg tea.Msg) (FilterPalette, FilterAction, tea.Cmd) {
	switch m.phase {
	case phaseTypeSelect:
		return m.updateTypeSelect(msg)
	case phaseColorSelect:
		return m.updateColorSelect(msg)
	case phaseCategorySelect:
		return m.updateCategorySelect(msg)
	}
	return m, FilterActionNone, nil
}

func (m FilterPalette) updateTypeSelect(msg tea.Msg) (FilterPalette, FilterAction, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, FilterActionNone, nil
	}
	switch {
	case key.Matches(km, filterKeys.Close):
		return m, FilterActionClose, nil

	case key.Matches(km, filterKeys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(km, filterKeys.Down):
		if m.cursor < len(filterTypeEntries)-1 {
			m.cursor++
		}

	case key.Matches(km, filterKeys.Select):
		entry := filterTypeEntries[m.cursor]
		switch {
		case entry.isToggle:
			return m, FilterActionToggle, nil

		case entry.clearAll:
			f := models.CardFilter{}
			return m, FilterActionApplied, func() tea.Msg { return FilterAppliedMsg{Filter: f} }

		default:
			m = m.enterPhase(entry.phase)
		}
	}
	return m, FilterActionNone, nil
}

func (m FilterPalette) enterPhase(phase filterPhase) FilterPalette {
	m.phase = phase
	m.cursor = 0

	switch phase {
	case phaseColorSelect:
		if len(m.current.Colors) == 1 {
			for i, e := range colorEntries {
				if e.color == m.current.Colors[0] {
					m.cursor = i
					break
				}
			}
		}
	case phaseCategorySelect:
		// Pre-select the current active category, accounting for FindPilots mode.
		activeCat := models.Category("")
		if m.current.FindPilots {
			activeCat = models.CategoryPilot
		} else if len(m.current.Categories) == 1 {
			activeCat = m.current.Categories[0]
		}
		if activeCat != "" {
			for i, e := range categoryEntries {
				if e.category == activeCat {
					m.cursor = i
					break
				}
			}
		}
	}
	return m
}

func (m FilterPalette) updateColorSelect(msg tea.Msg) (FilterPalette, FilterAction, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, FilterActionNone, nil
	}
	switch {
	case key.Matches(km, filterKeys.Close):
		m.phase = phaseTypeSelect
		m.cursor = 1 // back to "Filter by Color"

	case key.Matches(km, filterKeys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(km, filterKeys.Down):
		if m.cursor < len(colorEntries)-1 {
			m.cursor++
		}

	case key.Matches(km, filterKeys.Select):
		f := m.current
		if colorEntries[m.cursor].color == "" {
			f.Colors = nil
		} else {
			f.Colors = []models.Color{colorEntries[m.cursor].color}
		}
		return m, FilterActionApplied, func() tea.Msg { return FilterAppliedMsg{Filter: f} }
	}
	return m, FilterActionNone, nil
}

func (m FilterPalette) updateCategorySelect(msg tea.Msg) (FilterPalette, FilterAction, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, FilterActionNone, nil
	}
	switch {
	case key.Matches(km, filterKeys.Close):
		m.phase = phaseTypeSelect
		m.cursor = 2 // back to "Filter by Category"

	case key.Matches(km, filterKeys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(km, filterKeys.Down):
		if m.cursor < len(categoryEntries)-1 {
			m.cursor++
		}

	case key.Matches(km, filterKeys.Select):
		f := m.current
		cat := categoryEntries[m.cursor].category
		switch cat {
		case "":
			f.Categories = nil
			f.FindPilots = false
		case models.CategoryPilot:
			// Pilot filter covers both Pilot cards and Command cards with the Pilot trait.
			f.Categories = nil
			f.FindPilots = true
		default:
			f.Categories = []models.Category{cat}
			f.FindPilots = false
		}
		return m, FilterActionApplied, func() tea.Msg { return FilterAppliedMsg{Filter: f} }
	}
	return m, FilterActionNone, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m FilterPalette) View() string {
	var sb strings.Builder

	switch m.phase {
	case phaseTypeSelect:
		m.viewTypeSelect(&sb)
	case phaseColorSelect:
		m.viewList(&sb, "Select Color", colorListRows())
	case phaseCategorySelect:
		m.viewList(&sb, "Select Category", categoryListRows())
	}

	return styles.StylePaletteOverlay.Width(fOverlayW).Render(sb.String())
}

func (m FilterPalette) viewTypeSelect(sb *strings.Builder) {
	divider := styles.StyleDetailDivider.Render(strings.Repeat("─", fLabelW+fHintW))

	sb.WriteString(styles.StyleDetailTitle.Render("Filter Cards"))
	sb.WriteString("\n")
	sb.WriteString(divider)
	sb.WriteString("\n")

	for i, e := range filterTypeEntries {
		var label, hint string
		if e.isToggle {
			if m.filtersActive {
				label = "● Filters Active"
				hint = "enter to pause"
			} else {
				label = "○ Filters Paused"
				hint = "enter to activate"
			}
		} else {
			label = e.label
			hint = e.hint
		}

		row := lipgloss.NewStyle().Width(fLabelW).Render(label) +
			styles.StylePaletteHint.Width(fHintW).Render(hint)

		if i == m.cursor {
			sb.WriteString(styles.StylePaletteItemSelected.Render(row))
		} else {
			sb.WriteString(styles.StylePaletteItem.Render(row))
		}
		sb.WriteString("\n")

		if i == 0 {
			sb.WriteString(divider)
			sb.WriteString("\n")
		}
	}

	sb.WriteString(divider)
	sb.WriteString("\n")
	sb.WriteString(styles.StylePaletteItem.Render(
		styles.StylePaletteHint.Render("↑↓/jk · enter select · esc close"),
	))
}

func (m FilterPalette) viewList(sb *strings.Builder, title string, rows []listRow) {
	divider := styles.StyleDetailDivider.Render(strings.Repeat("─", fLabelW+fHintW))

	sb.WriteString(styles.StyleDetailTitle.Render(title))
	sb.WriteString("\n")
	sb.WriteString(divider)
	sb.WriteString("\n")

	for i, r := range rows {
		label := lipgloss.NewStyle().Width(fLabelW + fHintW).Render(r.label)
		if i == m.cursor {
			sb.WriteString(styles.StylePaletteItemSelected.Render(label))
		} else {
			sb.WriteString(styles.StylePaletteItem.Render(label))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(divider)
	sb.WriteString("\n")
	sb.WriteString(styles.StylePaletteItem.Render(
		styles.StylePaletteHint.Render("↑↓/jk · enter select · esc back"),
	))
}

type listRow struct{ label string }

func colorListRows() []listRow {
	rows := make([]listRow, len(colorEntries))
	for i, e := range colorEntries {
		var swatch string
		if e.color != "" {
			swatch = styles.CardColorSwatch(string(e.color)) + " "
		}
		rows[i] = listRow{label: swatch + e.label}
	}
	return rows
}

func categoryListRows() []listRow {
	rows := make([]listRow, len(categoryEntries))
	for i, e := range categoryEntries {
		rows[i] = listRow{label: e.label}
	}
	return rows
}
