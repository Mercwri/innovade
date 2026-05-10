package library

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Mercwri/innovade/internal/models"
	"github.com/Mercwri/innovade/internal/store"
	"github.com/Mercwri/innovade/internal/ui/keys"
)

// FilterPaletteModel is the interface used for the library's filter palette.
type FilterPaletteModel interface {
	Update(msg tea.Msg) (FilterPaletteModel, tea.Cmd)
	View() string
}

type filterPaletteModel struct {
	filter models.CardFilter
}

func NewFilterPaletteModel(filter models.CardFilter) FilterPaletteModel {
	return &filterPaletteModel{filter: filter}
}

func (m *filterPaletteModel) Update(msg tea.Msg) (FilterPaletteModel, tea.Cmd) {
	return m, nil
}

func (m *filterPaletteModel) View() string {
	return "Filter palette"
}

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

	// Sub-palette for library-specific actions
	filterPaletteOpen bool
	filterPalette     FilterPaletteModel

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

func New(s *store.Store) (Model, error) {
	m := Model{
		store:  s,
		filter: models.CardFilter{ExcludeTokens: false},
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

// OpenFilterPalette is called by the root app when the palette routes here.
func (m *Model) OpenFilterPalette() {
	m.filterPaletteOpen = true
	m.filterPalette = NewFilterPaletteModel(m.filter)
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
		m.filterPaletteOpen = false
		m.cursor = 0
		m.offset = 0
		return m, m.loadCards()

	case tea.KeyMsg:
		if m.filterPaletteOpen {
			var cmd tea.Cmd
			m.filterPalette, cmd = m.filterPalette.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
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
			m.moveCursor(-m.listH)
		case key.Matches(msg, keys.Library.PageDown):
			m.moveCursor(m.listH)
		}
	}

	return m, tea.Batch(cmds...)
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

func (m *Model) scrollToSelected() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.listH {
		m.offset = m.cursor - m.listH + 1
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
	return func() tea.Msg {
		q := models.CardQuery{
			Filter: m.filter,
			SortBy: models.SortByCardCode,
			Order:  models.SortAsc,
		}
		cards, err := m.store.QueryCards(q)
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
		return renderWithFilterPalette(list, detail, m.filterPalette.View(), m.width, m.height)
	}

	return renderLayout(list, detail, m.width, m.height)
}
