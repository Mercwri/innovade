package ui

import (
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Mercwri/innovade/internal/models"
	"github.com/Mercwri/innovade/internal/store"
	"github.com/Mercwri/innovade/internal/ui/analysis"
	"github.com/Mercwri/innovade/internal/ui/deckbuilder"
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
	Quit        key.Binding
	Palette     key.Binding
	GoLibrary   key.Binding
	GoDeckBuild key.Binding
}{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("q", "quit"),
	),
	Palette: key.NewBinding(
		key.WithKeys("?", "ctrl+p"),
		key.WithHelp("?", "palette"),
	),
	GoLibrary: key.NewBinding(
		key.WithKeys("!"),
		key.WithHelp("⇧1", "library"),
	),
	GoDeckBuild: key.NewBinding(
		key.WithKeys("@"),
		key.WithHelp("⇧2", "deck builder"),
	),
}

// AppModel is the root Bubble Tea model. It owns the active view and
// renders the command palette as a full-screen overlay when open.
type AppModel struct {
	store  *store.Store
	width  int
	height int

	activeView  View
	library     library.Model
	deckbuilder deckbuilder.Model
	analysis    analysis.Model

	activeDeck  *models.Deck
	limitations []models.LimitationRule

	paletteOpen bool
	palette     palette.PaletteModel
}

// New constructs the root application model.
func New(s *store.Store) (AppModel, error) {
	lib, err := library.New(s)
	if err != nil {
		return AppModel{}, err
	}

	limitations, err := models.LoadLimitationsFromPath(filepath.Join("data", "bans.json"))
	if err != nil {
		limitations = nil
	}

	return AppModel{
		store:       s,
		activeView:  ViewLibrary,
		library:     lib,
		deckbuilder: deckbuilder.New(s),
		analysis:    analysis.New(s),
		limitations: limitations,
	}, nil
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(m.library.Init(), m.deckbuilder.Init())
}

func (m AppModel) validateDeck(deck *models.Deck) models.ValidationResult {
	if len(m.limitations) == 0 {
		return models.ValidationResult{Valid: true}
	}

	codes := make([]string, 0, len(deck.Entries))
	seen := make(map[string]struct{}, len(deck.Entries))
	for _, entry := range deck.Entries {
		if entry.Quantity <= 0 {
			continue
		}
		if _, ok := seen[entry.CardCode]; ok {
			continue
		}
		seen[entry.CardCode] = struct{}{}
		codes = append(codes, entry.CardCode)
	}

	cards := map[string]models.Card{}
	if len(codes) > 0 {
		loaded, err := m.store.GetCardsByCodeMap(codes)
		if err == nil {
			cards = loaded
		}
	}

	return models.ValidateDeck(m.limitations, deck, cards)
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
		m.deckbuilder.SetSize(msg.Width, msg.Height-1)
		m.analysis.SetSize(msg.Width, msg.Height-1)
		return m, nil

	case deckbuilder.DeckSelectedMsg:
		m.activeDeck = msg.Deck
		m.library.SetActiveDeck(msg.Deck)
		m.activeView = ViewLibrary
		if colors := deckColors(msg.Deck, m.deckbuilder.CardStats()); len(colors) > 0 {
			m.library.SetFilter(models.CardFilter{Colors: colors})
		}
		return m, tea.Batch(
			m.library.ReloadCards(),
			saveSessionCmd(m.store, "active_deck_id", msg.Deck.ID),
		)

	case deckbuilder.DeckCreatedMsg:
		m.activeDeck = msg.Deck
		m.library.SetActiveDeck(msg.Deck)
		m.activeView = ViewLibrary
		return m, tea.Batch(
			saveDeckCmd(m.store, msg.Deck),
			saveSessionCmd(m.store, "active_deck_id", msg.Deck.ID),
		)

	case library.AddToDeckMsg:
		if m.activeDeck != nil {
			if m.activeDeck.CardCount(msg.CardCode) >= models.MaxCopiesPerCard {
				m.deckbuilder.SetStatus("max copies already reached", true)
				return m, nil
			}

			// Entries must be deep-copied: Deck.AddCard mutates an existing
			// entry's Quantity in place, and a shallow struct copy still
			// shares the same backing array, so validating here would
			// silently bump the real deck's quantity before AddCard below
			// bumps it again (1 press → +2).
			proposed := *m.activeDeck
			proposed.Entries = append([]models.DeckEntry(nil), m.activeDeck.Entries...)
			proposed.AddCard(msg.CardCode)
			if result := m.validateDeck(&proposed); !result.Valid {
				m.deckbuilder.SetStatus("deck would violate current bans/restrictions", true)
				return m, nil
			}

			isNew := m.activeDeck.CardCount(msg.CardCode) == 0
			m.activeDeck.AddCard(msg.CardCode)
			m.deckbuilder.SetStatus("", false)
			if isNew {
				m.library.SetActiveDeck(m.activeDeck) // new card → re-sort, cursor to top
			} else {
				m.library.RefreshActiveDeck(m.activeDeck) // quantity bump → stay put
			}
			return m, saveDeckCmd(m.store, m.activeDeck)
		}

	case library.RemoveFromDeckMsg:
		if m.activeDeck != nil {
			willLeave := m.activeDeck.CardCount(msg.CardCode) <= 1
			m.activeDeck.RemoveCard(msg.CardCode)
			m.deckbuilder.SetStatus("", false)
			if willLeave {
				m.library.SetActiveDeck(m.activeDeck) // card gone → re-sort, cursor to top
			} else {
				m.library.RefreshActiveDeck(m.activeDeck) // quantity drop → stay put
			}
			return m, saveDeckCmd(m.store, m.activeDeck)
		}

	case deckbuilder.DeckDeletedMsg:
		if m.activeDeck != nil && m.activeDeck.ID == msg.ID {
			m.activeDeck = nil
			m.library.SetActiveDeck(nil)
			cmds = append(cmds, saveSessionCmd(m.store, "active_deck_id", ""))
		}
		return m, tea.Batch(append(cmds, m.deckbuilder.Reload())...)

	case tea.KeyMsg:
		// When the deckbuilder is capturing text input, bypass all global
		// hotkeys so characters like '?' don't open the palette.
		if m.activeView == ViewDeckBuilder && m.deckbuilder.Capturing() {
			var cmd tea.Cmd
			m.deckbuilder, cmd = m.deckbuilder.Update(msg)
			return m, cmd
		}

		// Global quit — save session then exit
		if key.Matches(msg, Global.Quit) {
			if m.activeDeck != nil {
				_ = m.store.SetSessionValue("active_deck_id", m.activeDeck.ID)
			}
			return m, tea.Quit
		}

		// View-switch hotkeys (active even when palette is open)
		if !m.paletteOpen {
			if key.Matches(msg, Global.GoLibrary) {
				m.activeView = ViewLibrary
				return m, nil
			}
			if key.Matches(msg, Global.GoDeckBuild) {
				m.activeView = ViewDeckBuilder
				return m, m.deckbuilder.Reload()
			}
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

			switch action {
			case palette.ActionClose:
				m.paletteOpen = false
			case palette.ActionNone:
				// keep palette open
			default:
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
		case ViewDeckBuilder:
			var cmd tea.Cmd
			m.deckbuilder, cmd = m.deckbuilder.Update(msg)
			cmds = append(cmds, cmd)
		case ViewAnalysis:
			var cmd tea.Cmd
			m.analysis, cmd = m.analysis.Update(msg)
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
	case ViewDeckBuilder:
		body = m.deckbuilder.View()
	case ViewAnalysis:
		body = m.analysis.View()
	default:
		body = "Coming soon"
	}

	base := lipgloss.JoinVertical(lipgloss.Left, header, body)

	if m.paletteOpen {
		return renderWithOverlay(base, m.palette.View(), m.width, m.height)
	}

	if m.width == 0 || m.height == 0 {
		return base
	}
	return styles.StyleBase.Width(m.width).Height(m.height).Render(base)
}

func (m AppModel) renderHeader() string {
	left := styles.StyleHeader.Render("INNOVADE")
	right := styles.StyleHeaderMuted.Render("[⇧1] library  [⇧2] decks  [?] palette")

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)

	// left+right must never exceed m.width: if they did, the joined header
	// would be wider than the terminal, causing it to soft-wrap onto a
	// second row. Every view sizes its body as m.height-1 assuming a
	// single-row header, so that wrap pushes the body's last row off
	// screen — the "top bar clips out" symptom. Hints truncate first,
	// the app name only as a last resort on very narrow terminals.
	if leftWidth+rightWidth > m.width {
		right = styles.StyleHeaderMuted.MaxWidth(max(m.width-leftWidth, 0)).
			Render("[⇧1] library  [⇧2] decks  [?] palette")
		rightWidth = lipgloss.Width(right)
	}
	if leftWidth+rightWidth > m.width {
		left = styles.StyleHeader.MaxWidth(m.width).Render("INNOVADE")
		leftWidth = lipgloss.Width(left)
	}

	centerWidth := max(m.width-leftWidth-rightWidth, 0)
	center := styles.StyleHeader.Width(centerWidth).Align(lipgloss.Center).Render(viewLabel(m.activeView))

	return lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)
}

type importResultMsg struct {
	imported int
	err      error
}

func saveSessionCmd(s *store.Store, key, value string) tea.Cmd {
	return func() tea.Msg {
		_ = s.SetSessionValue(key, value)
		return nil
	}
}

func saveDeckCmd(s *store.Store, d *models.Deck) tea.Cmd {
	return func() tea.Msg {
		_ = s.SaveDeck(d)
		return nil
	}
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
		return m, m.deckbuilder.Reload()
	case palette.ActionOpenDeckLibrary:
		m.activeView = ViewDeckLibrary
	case palette.ActionOpenAnalysis:
		m.activeView = ViewAnalysis
		return m, m.analysis.SetDeck(m.activeDeck)
	case palette.ActionQuit:
		return m, tea.Quit
	case palette.ActionFilterLibrary:
		return m, m.library.OpenFilterPalette()
	case palette.ActionImportCards:
		return m, importCardsCmd(m.store)
	}
	return m, nil
}

// renderWithOverlay centers the palette in the upper third of the screen.
func renderWithOverlay(_ string, overlay string, w, h int) string {
	overlayH := lipgloss.Height(overlay)
	y := max((h-overlayH)/4, 0)
	return lipgloss.Place(w, h,
		lipgloss.Center, lipgloss.Top,
		lipgloss.NewStyle().MarginTop(y).Render(overlay),
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#111827")),
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#111827")),
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

// deckColors returns the unique colors present across all cards in the deck,
// derived from cached card stats. Returns at most the distinct colors found.
func deckColors(deck *models.Deck, stats map[string]store.CardStat) []models.Color {
	seen := make(map[models.Color]bool)
	var result []models.Color
	for _, e := range deck.Entries {
		if s, ok := stats[e.CardCode]; ok {
			for _, c := range s.Colors {
				if !seen[c] {
					seen[c] = true
					result = append(result, c)
				}
			}
		}
	}
	return result
}

// ClosePaletteMsg is sent by child models to close the palette.
type ClosePaletteMsg struct{}
