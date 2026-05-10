package ui

import (
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Mercwri/innovade/internal/store"
	"github.com/Mercwri/innovade/internal/ui/library"
	"github.com/Mercwri/innovade/internal/ui/palette"
	"github.com/Mercwri/innovade/internal/ui/styles"
)

// View identifies which feature is currently active.
type View int

const (
	ViewLibrary View = iota
	ViewDeckBuilder
	ViewDeckLibrary
	ViewAnalysis
)

var Global = struct {
	Quit    key.Binding
	Palette key.Binding
}{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("q", "quit"),
	),
	Palette: key.NewBinding(
		key.WithKeys("?", "ctrl+p"),
		key.WithHelp("?", "palette"),
	),
}

// AppModel is the root Bubble Tea model. It owns the active view and
// renders the command palette as a full-screen overlay when open.
type AppModel struct {
	store  *store.Store
	width  int
	height int

	activeView View
	library    library.Model

	paletteOpen bool
	palette     palette.PaletteModel
}

// New constructs the root application model.
func New(s *store.Store) (AppModel, error) {
	lib, err := library.New(s)
	if err != nil {
		return AppModel{}, err
	}

	return AppModel{
		store:      s,
		activeView: ViewLibrary,
		library:    lib,
	}, nil
}

func (m AppModel) Init() tea.Cmd {
	return m.library.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case importResultMsg:
		if msg.err != nil {
			m.library.SetError(msg.err)
			return m, nil
		}
		return m, m.library.ReloadCards()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.library.SetSize(msg.Width, msg.Height-1) // -1 for header
		return m, nil

	case tea.KeyMsg:
		// Global quit — always active
		if key.Matches(msg, Global.Quit) {
			return m, tea.Quit
		}

		// Palette toggle
		if !m.paletteOpen && key.Matches(msg, Global.Palette) {
			m.paletteOpen = true
			m.palette = palette.NewPaletteModel(palette.View(m.activeView))
			return m, nil
		}

		// Route to palette when open
		if m.paletteOpen {
			var cmd tea.Cmd
			var action palette.PaletteAction
			m.palette, action, cmd = m.palette.Update(msg)
			cmds = append(cmds, cmd)

			if action != palette.ActionNone {
				m.paletteOpen = false
				m, cmd = m.handlePaletteAction(action)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

	case ClosePaletteMsg:
		m.paletteOpen = false
		return m, nil
	}

	// Route to the active view
	if !m.paletteOpen {
		switch m.activeView {
		case ViewLibrary:
			var cmd tea.Cmd
			m.library, cmd = m.library.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) View() string {
	header := m.renderHeader()
	var body string

	switch m.activeView {
	case ViewLibrary:
		body = m.library.View()
	default:
		body = "Coming soon"
	}

	base := lipgloss.JoinVertical(lipgloss.Left, header, body)

	if m.paletteOpen {
		return renderWithOverlay(base, m.palette.View(), m.width, m.height)
	}

	return base
}

func (m AppModel) renderHeader() string {
	left := styles.StyleHeader.Render("INNOVADE")
	center := styles.StyleHeader.Render(viewLabel(m.activeView))
	right := styles.StyleHeaderMuted.Render("[?] palette")

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(center) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	spacer := lipgloss.NewStyle().Width(gap / 2).Render("")
	return lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, center, spacer, right)
}

type importResultMsg struct {
	imported int
	err      error
}

func importCardsCmd(s *store.Store) tea.Cmd {
	return func() tea.Msg {
		imported, err := s.ImportCardsFromJSON(filepath.Join("data", "cards.json"))
		return importResultMsg{imported: imported, err: err}
	}
}

func (m AppModel) handlePaletteAction(action palette.PaletteAction) (AppModel, tea.Cmd) {
	switch action {
	case palette.ActionOpenLibrary:
		m.activeView = ViewLibrary
	case palette.ActionOpenDeckBuilder:
		m.activeView = ViewDeckBuilder
	case palette.ActionOpenDeckLibrary:
		m.activeView = ViewDeckLibrary
	case palette.ActionOpenAnalysis:
		m.activeView = ViewAnalysis
	case palette.ActionQuit:
		return m, tea.Quit
	case palette.ActionFilterLibrary:
		m.library.OpenFilterPalette()
	case palette.ActionImportCards:
		return m, importCardsCmd(m.store)
	}
	return m, nil
}

// renderWithOverlay centers the palette view on top of the base content.
func renderWithOverlay(base, overlay string, w, h int) string {
	overlayW := lipgloss.Width(overlay)
	overlayH := lipgloss.Height(overlay)

	x := (w - overlayW) / 2
	y := (h - overlayH) / 4 // place in upper third

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	return lipgloss.Place(w, h,
		lipgloss.Left, lipgloss.Top,
		lipgloss.NewStyle().
			MarginLeft(x).
			MarginTop(y).
			Render(overlay),
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#111827")),
	)
}

func viewLabel(v View) string {
	switch v {
	case ViewLibrary:
		return "Card Library"
	case ViewDeckBuilder:
		return "Deck Builder"
	case ViewDeckLibrary:
		return "Deck Library"
	case ViewAnalysis:
		return "Analysis"
	}
	return ""
}

// ClosePaletteMsg is sent by child models to close the palette.
type ClosePaletteMsg struct{}
