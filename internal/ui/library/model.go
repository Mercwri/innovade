package library

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Mercwri/innovade/internal/models"
	"github.com/Mercwri/innovade/internal/store"
	"github.com/Mercwri/innovade/internal/ui/keys"
)

// Model is the card library Bubble Tea model.
type Model struct {
	store *store.Store

	// Data
	cards    []models.Card
	total    int
	filter   models.CardFilter
	selected *models.Card

	// List state
	cursor int
	offset int // first visible row index
	listH  int // visible rows in list
	width  int
	height int

	// Filtering state
	filtersActive     bool
	filterPaletteOpen bool
	filterPalette     FilterPalette

	// Deck editing state
	activeDeck *models.Deck

	loaded bool
	err    error
}

// CardsLoadedMsg is sent when the initial card fetch completes.
type CardsLoadedMsg struct {
	Cards []models.Card
	Err   error
}

// FilterAppliedMsg is sent when the filter palette resolves a new filter.
type FilterAppliedMsg struct {
	Filter models.CardFilter
}

// AddToDeckMsg is sent when the user wants to add the selected card to the active deck.
type AddToDeckMsg struct{ CardCode string }

// RemoveFromDeckMsg is sent when the user wants to remove the selected card from the active deck.
type RemoveFromDeckMsg struct{ CardCode string }

func New(s *store.Store) (Model, error) {
	m := Model{
		store:         s,
		filter:        models.CardFilter{ExcludeTokens: false},
		filtersActive: true,
	}
	return m, nil
}

func (m Model) Init() tea.Cmd {
	return m.ReloadCards()
}

func (m *Model) ReloadCards() tea.Cmd {
	m.loaded = false
	m.err = nil
	return m.loadCards()
}

func (m *Model) SetError(err error) {
	m.err = err
	m.loaded = true
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	// Reserve bottom portion for detail panel
	m.listH = h * 2 / 3
}

// SetActiveDeck updates the deck being edited in the library view.
func (m *Model) SetActiveDeck(d *models.Deck) {
	m.activeDeck = d
}

// OpenFilterPalette is called by the root app when the palette routes here.
func (m *Model) OpenFilterPalette() {
	m.filterPaletteOpen = true
	m.filterPalette = NewFilterPalette(m.filter, m.filtersActive)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case CardsLoadedMsg:
		m.loaded = true
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.cards = msg.Cards
		m.total = len(msg.Cards)
		m.cursor = 0
		m.offset = 0
		m.updateSelected()
		return m, nil

	case FilterAppliedMsg:
		m.filter = msg.Filter
		m.filtersActive = true
		m.filterPaletteOpen = false
		m.cursor = 0
		m.offset = 0
		return m, m.loadCards()

	case tea.KeyMsg:
		if m.filterPaletteOpen {
			var action FilterAction
			var cmd tea.Cmd
			m.filterPalette, action, cmd = m.filterPalette.Update(msg)
			cmds = append(cmds, cmd)
			switch action {
			case FilterActionClose, FilterActionApplied:
				m.filterPaletteOpen = false
			case FilterActionToggle:
				m.filtersActive = !m.filtersActive
				m.filterPaletteOpen = false
				cmds = append(cmds, m.loadCards())
			}
			return m, tea.Batch(cmds...)
		}

		// Deck add/remove — only when a deck is active
		if m.activeDeck != nil && m.selected != nil {
			switch {
			case key.Matches(msg, keys.Library.AddToDeck):
				code := m.selected.CardCode
				return m, func() tea.Msg { return AddToDeckMsg{CardCode: code} }
			case key.Matches(msg, keys.Library.RemoveFromDeck):
				code := m.selected.CardCode
				return m, func() tea.Msg { return RemoveFromDeckMsg{CardCode: code} }
			}
		}

		switch {
		case key.Matches(msg, keys.Library.Up):
			m.moveCursor(-1)
		case key.Matches(msg, keys.Library.Down):
			m.moveCursor(1)
		case key.Matches(msg, keys.Library.Top):
			m.cursor = 0
			m.offset = 0
			m.updateSelected()
		case key.Matches(msg, keys.Library.Bottom):
			m.cursor = len(m.cards) - 1
			m.scrollToSelected()
			m.updateSelected()
		case key.Matches(msg, keys.Library.PageUp):
			m.moveCursor(-m.visibleRows())
		case key.Matches(msg, keys.Library.PageDown):
			m.moveCursor(m.visibleRows())
		case key.Matches(msg, keys.Library.FindLinks):
			if m.selected != nil {
				if f, ok := buildLinkFilter(m.selected); ok {
					return m, func() tea.Msg { return FilterAppliedMsg{Filter: f} }
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// buildLinkFilter constructs a CardFilter that finds cards linked to the given card.
// For a Unit: finds pilot-eligible cards matching its link requirement terms.
// For a Pilot/pilot-Command: finds Units whose link_requirement matches the card's name/traits.
func buildLinkFilter(card *models.Card) (models.CardFilter, bool) {
	switch {
	case card.Category == models.CategoryUnit || card.Category == models.CategoryUnitToken:
		terms := card.ParseLinkRequirement()
		if len(terms) == 0 {
			return models.CardFilter{}, false
		}
		return models.CardFilter{
			FindPilots:     true,
			PilotLinkTerms: terms,
			Description:    "links for " + card.CardCode,
		}, true

	case card.IsPilotEligible():
		terms := []string{card.Name}
		for _, t := range card.Types {
			if !strings.EqualFold(t, "pilot") {
				terms = append(terms, t)
			}
		}
		return models.CardFilter{
			Categories:    []models.Category{models.CategoryUnit},
			UnitLinkTerms: terms,
			Description:   "units for " + card.Name,
		}, true
	}
	return models.CardFilter{}, false
}

func (m *Model) moveCursor(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.cards) {
		m.cursor = len(m.cards) - 1
	}
	m.scrollToSelected()
	m.updateSelected()
}

// visibleRows returns the number of card rows that fit in the list area,
// reserving space for the 3 fixed header rows and 1 scroll-hint row.
func (m Model) visibleRows() int {
	v := m.listH - 4
	if v < 0 {
		return 0
	}
	return v
}

func (m *Model) scrollToSelected() {
	vr := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+vr {
		m.offset = m.cursor - vr + 1
	}
}

func (m *Model) updateSelected() {
	if len(m.cards) == 0 {
		m.selected = nil
		return
	}
	m.selected = &m.cards[m.cursor]
}

func (m *Model) loadCards() tea.Cmd {
	filter := models.CardFilter{}
	if m.filtersActive {
		filter = m.filter
	}
	store := m.store
	return func() tea.Msg {
		q := models.CardQuery{
			Filter: filter,
			SortBy: models.SortByCardCode,
			Order:  models.SortAsc,
		}
		cards, err := store.QueryCards(q)
		return CardsLoadedMsg{Cards: cards, Err: err}
	}
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("error loading cards: %v", m.err)
	}
	if !m.loaded {
		return "Loading cards..."
	}
	if len(m.cards) == 0 {
		return "No cards imported yet.\n\nPress [?] to open the palette and import cards."
	}

	list := renderList(m)
	detail := renderDetail(m.selected, m.width)

	if m.filterPaletteOpen {
		return renderWithFilterPalette(m.filterPalette.View(), m.width, m.height)
	}

	return renderLayout(list, detail, m.width, m.height)
}
