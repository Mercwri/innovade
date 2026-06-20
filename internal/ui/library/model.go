package library

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	listW  int // width of left list panel
	width  int
	height int

	// Filtering state
	filtersActive     bool
	filterPaletteOpen bool
	filterPalette     FilterPalette

	// Text-search input state
	textInputOpen bool
	textInput     string

	// Deck editing state
	activeDeck *models.Deck

	// Art
	imageDir      string          // local directory for cached card images
	downloadingArt map[string]bool // card codes currently being fetched

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

// SetsLoadedMsg is sent when the set list fetch completes.
type SetsLoadedMsg struct {
	Sets []models.CardSet
	Err  error
}

// AddToDeckMsg is sent when the user wants to add the selected card to the active deck.
type AddToDeckMsg struct{ CardCode string }

// RemoveFromDeckMsg is sent when the user wants to remove the selected card from the active deck.
type RemoveFromDeckMsg struct{ CardCode string }

// ImageReadyMsg is sent when a card image has been downloaded to disk.
type ImageReadyMsg struct{ CardCode string }

func New(s *store.Store) (Model, error) {
	m := Model{
		store:          s,
		filter:         models.CardFilter{ExcludeTokens: false},
		filtersActive:  true,
		downloadingArt: make(map[string]bool),
	}
	if dir, err := store.DataDir(); err == nil {
		m.imageDir = filepath.Join(dir, "images")
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
	m.listH = h
	m.listW = w * 55 / 100
}

// SetActiveDeck updates the deck being edited in the library view and
// re-sorts the current card list so deck cards appear first.
func (m *Model) SetActiveDeck(d *models.Deck) {
	m.activeDeck = d
	if d != nil && m.loaded && len(m.cards) > 0 {
		sortDeckCardsFirst(m.cards, d)
		m.cursor = 0
		m.offset = 0
		m.updateSelected() // download cmd discarded; deck switch doesn't need it
	}
}

// RefreshActiveDeck updates the deck pointer without re-sorting or resetting the cursor.
// Use when a card's quantity changes but it hasn't entered or left the deck.
func (m *Model) RefreshActiveDeck(d *models.Deck) {
	m.activeDeck = d
}

// SetFilter replaces the active filter without triggering a reload.
// The caller is responsible for calling ReloadCards() if needed.
func (m *Model) SetFilter(f models.CardFilter) {
	m.filter = f
	m.filtersActive = true
	m.cursor = 0
	m.offset = 0
}

// OpenFilterPalette is called by the root app when the palette routes here.
func (m *Model) OpenFilterPalette() tea.Cmd {
	m.filterPaletteOpen = true
	m.filterPalette = NewFilterPalette(m.filter, m.filtersActive)
	s := m.store
	return func() tea.Msg {
		sets, err := s.GetAllSets()
		return SetsLoadedMsg{Sets: sets, Err: err}
	}
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
		if m.activeDeck != nil {
			sortDeckCardsFirst(m.cards, m.activeDeck)
		}
		m.cursor = 0
		m.offset = 0
		cmds = append(cmds, m.updateSelected())
		return m, tea.Batch(cmds...)

	case ImageReadyMsg:
		delete(m.downloadingArt, msg.CardCode)
		return m, nil

	case SetsLoadedMsg:
		if msg.Err == nil {
			m.filterPalette.sets = msg.Sets
		}
		return m, nil

	case FilterAppliedMsg:
		m.filter = msg.Filter
		m.filtersActive = true
		m.filterPaletteOpen = false
		m.cursor = 0
		m.offset = 0
		return m, m.loadCards()

	case tea.KeyMsg:
		if m.textInputOpen {
			return m.updateTextInput(msg)
		}

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
			cmds = append(cmds, m.moveCursor(-1))
		case key.Matches(msg, keys.Library.Down):
			cmds = append(cmds, m.moveCursor(1))
		case key.Matches(msg, keys.Library.Top):
			m.cursor = 0
			m.offset = 0
			cmds = append(cmds, m.updateSelected())
		case key.Matches(msg, keys.Library.Bottom):
			m.cursor = len(m.cards) - 1
			m.scrollToSelected()
			cmds = append(cmds, m.updateSelected())
		case key.Matches(msg, keys.Library.PageUp):
			cmds = append(cmds, m.moveCursor(-m.visibleRows()))
		case key.Matches(msg, keys.Library.PageDown):
			cmds = append(cmds, m.moveCursor(m.visibleRows()))
		case key.Matches(msg, keys.Library.FindLinks):
			if m.selected != nil {
				if f, ok := buildLinkFilter(m.selected); ok {
					return m, func() tea.Msg { return FilterAppliedMsg{Filter: f} }
				}
			}
		case key.Matches(msg, keys.Library.ClearFilters):
			m.filter = models.CardFilter{}
			m.filtersActive = true
			m.cursor = 0
			m.offset = 0
			return m, m.loadCards()
		case key.Matches(msg, keys.Library.OpenFilter):
			cmds = append(cmds, m.OpenFilterPalette())
		case key.Matches(msg, keys.Library.TextFilter):
			m.textInputOpen = true
			m.textInput = m.filter.Name
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateTextInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		f := m.filter
		f.Name = strings.TrimSpace(m.textInput)
		m.filter = f
		m.filtersActive = true
		m.textInputOpen = false
		m.cursor = 0
		m.offset = 0
		return m, m.loadCards()
	case tea.KeyEscape:
		m.textInputOpen = false
		m.textInput = m.filter.Name
	case tea.KeyBackspace, tea.KeyDelete:
		runes := []rune(m.textInput)
		if len(runes) > 0 {
			m.textInput = string(runes[:len(runes)-1])
		}
	case tea.KeyRunes:
		m.textInput += string(msg.Runes)
	}
	return m, nil
}

func sortDeckCardsFirst(cards []models.Card, deck *models.Deck) {
	inDeck := make(map[string]bool, len(deck.Entries))
	for _, e := range deck.Entries {
		if e.Quantity > 0 {
			inDeck[e.CardCode] = true
		}
	}
	sort.SliceStable(cards, func(i, j int) bool {
		return inDeck[cards[i].CardCode] && !inDeck[cards[j].CardCode]
	})
}

// buildLinkFilter constructs a CardFilter that finds cards linked to the given card.
// For a Unit: finds pilot-eligible cards matching its link requirement terms.
// For a Pilot/pilot-Command: finds Units whose link_requirement references this card.
func buildLinkFilter(card *models.Card) (models.CardFilter, bool) {
	switch {
	case card.Category == models.CategoryUnit || card.Category == models.CategoryUnitToken:
		names, traits := card.ParseLinkTerms()
		if len(names) == 0 && len(traits) == 0 {
			return models.CardFilter{}, false
		}
		return models.CardFilter{
			FindPilots:      true,
			PilotLinkNames:  names,
			PilotLinkTraits: traits,
			Description:     "links for " + card.CardCode,
		}, true

	case card.IsPilotEligible():
		var unitTraits []string
		for _, t := range card.Types {
			if !strings.EqualFold(t, "pilot") {
				unitTraits = append(unitTraits, t)
			}
		}
		// Command-Pilot cards link by pilot name (from effect), not by card name.
		linkName := card.Name
		if card.Category == models.CategoryCommand {
			if pn := card.ExtractPilotName(); pn != "" {
				linkName = pn
			}
		}
		return models.CardFilter{
			Categories:     []models.Category{models.CategoryUnit},
			UnitLinkNames:  []string{linkName},
			UnitLinkTraits: unitTraits,
			Description:    "units for " + linkName,
		}, true
	}
	return models.CardFilter{}, false
}

func (m *Model) moveCursor(delta int) tea.Cmd {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.cards) {
		m.cursor = len(m.cards) - 1
	}
	m.scrollToSelected()
	return m.updateSelected()
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

// updateSelected sets m.selected and returns a download command if the card's
// image isn't cached locally yet.
func (m *Model) updateSelected() tea.Cmd {
	if len(m.cards) == 0 {
		m.selected = nil
		return nil
	}
	m.selected = &m.cards[m.cursor]
	return m.maybeDownloadArt(m.selected)
}

// maybeDownloadArt returns a Cmd that downloads the card's default image if it
// isn't present on disk and isn't already being fetched.
func (m *Model) maybeDownloadArt(card *models.Card) tea.Cmd {
	if m.imageDir == "" || card == nil || card.DefaultImagePath == "" {
		return nil
	}
	if m.downloadingArt[card.CardCode] {
		return nil
	}
	localPath := filepath.Join(m.imageDir, card.DefaultImagePath)
	if _, err := os.Stat(localPath); err == nil {
		return nil // already on disk
	}

	m.downloadingArt[card.CardCode] = true
	cardCode := card.CardCode
	filename := card.DefaultImagePath
	return func() tea.Msg {
		_ = downloadArt(localPath, filename) // failure is silent; placeholder shows instead
		return ImageReadyMsg{CardCode: cardCode}
	}
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

func hasAnyFilter(f models.CardFilter) bool {
	return f.Name != "" || len(f.Categories) > 0 || len(f.Colors) > 0 ||
		f.FindPilots || len(f.PilotLinkNames) > 0 || len(f.PilotLinkTraits) > 0 ||
		len(f.UnitLinkNames) > 0 || len(f.UnitLinkTraits) > 0 ||
		len(f.SetCodes) > 0 || f.Type != ""
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("error loading cards: %v", m.err)
	}
	if !m.loaded {
		return "Loading cards..."
	}
	if len(m.cards) == 0 {
		if m.filtersActive && hasAnyFilter(m.filter) {
			return "No cards match the current filter.\n\nPress [c] to clear filters or [f] to adjust them."
		}
		return "No cards imported yet.\n\nPress [?] to open the palette and import cards."
	}

	var imagePath string
	if m.selected != nil && m.selected.DefaultImagePath != "" && m.imageDir != "" {
		imagePath = filepath.Join(m.imageDir, m.selected.DefaultImagePath)
	}

	detailW := m.width - m.listW
	list := renderList(m)
	detail := renderDetail(m.selected, detailW, m.height, imagePath)

	if m.textInputOpen {
		return renderWithTextInput(m.textInput, m.width, m.height)
	}
	if m.filterPaletteOpen {
		return renderWithFilterPalette(m.filterPalette.View(), m.width, m.height)
	}

	return renderLayout(list, detail, m.listW, m.width, m.height)
}
