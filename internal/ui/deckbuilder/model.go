package deckbuilder

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Mercwri/innovade/internal/models"
	"github.com/Mercwri/innovade/internal/store"
	"github.com/Mercwri/innovade/internal/ui/styles"
)

var dbKeys = struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	New    key.Binding
	Delete key.Binding
	Export key.Binding
	Import key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up", "k")),
	Down:   key.NewBinding(key.WithKeys("down", "j")),
	Select: key.NewBinding(key.WithKeys("enter")),
	New:    key.NewBinding(key.WithKeys("n")),
	Delete: key.NewBinding(key.WithKeys("x")),
	Export: key.NewBinding(key.WithKeys("e")),
	Import: key.NewBinding(key.WithKeys("i")),
}

// DeckSelectedMsg is sent when the user activates an existing deck for editing in the library.
type DeckSelectedMsg struct{ Deck *models.Deck }

// DeckCreatedMsg is sent when the user creates a new deck.
type DeckCreatedMsg struct{ Deck *models.Deck }

// DeckDeletedMsg is sent after a deck is removed from the store.
type DeckDeletedMsg struct{ ID string }

type decksLoadedMsg struct {
	decks     []models.Deck
	cardNames map[string]string
	err       error
}

// Model is the deck builder Bubble Tea model.
type Model struct {
	store     *store.Store
	decks     []models.Deck
	cardNames map[string]string
	cursor    int
	width     int
	height    int
	loaded    bool
	err       error

	status    string
	statusErr bool

	importOpen    bool
	importPalette ImportPalette
}

func New(s *store.Store) Model {
	return Model{store: s, cardNames: map[string]string{}}
}

func (m Model) Init() tea.Cmd {
	return m.loadDecks()
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m Model) Reload() tea.Cmd {
	return m.loadDecks()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case decksLoadedMsg:
		m.loaded = true
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.decks = msg.decks
		if msg.cardNames != nil {
			m.cardNames = msg.cardNames
		}
		// clamp cursor so it stays on a valid row after deletions
		if m.cursor > len(m.decks) {
			m.cursor = len(m.decks)
		}
		return m, nil

	case exportResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("export failed: %v", msg.err)
			m.statusErr = true
		} else {
			m.status = fmt.Sprintf("Exported → %s", msg.path)
			m.statusErr = false
		}
		return m, nil

	case importResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("import failed: %v", msg.err)
			m.statusErr = true
		} else {
			m.status = fmt.Sprintf("Imported '%s'", msg.deckName)
			m.statusErr = false
			return m, m.loadDecks()
		}
		return m, nil

	case tea.KeyMsg:
		// Import palette captures keys while open — do not clear status.
		if m.importOpen {
			var action importAction
			m.importPalette, action = m.importPalette.Update(msg)
			switch action {
			case importActionClose:
				m.importOpen = false
			case importActionSelect:
				file := m.importPalette.selectedFile()
				m.importOpen = false
				return m, importDeckCmd(m.store, file)
			}
			return m, nil
		}

		m.status = "" // clear status on main-view keypresses
		total := 1 + len(m.decks) // "New Deck" sentinel + saved decks
		switch {
		case key.Matches(msg, dbKeys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, dbKeys.Down):
			if m.cursor < total-1 {
				m.cursor++
			}
		case key.Matches(msg, dbKeys.New):
			d := &models.Deck{Name: "no name"}
			return m, func() tea.Msg { return DeckCreatedMsg{Deck: d} }
		case key.Matches(msg, dbKeys.Select):
			if m.cursor == 0 {
				d := &models.Deck{Name: "no name"}
				return m, func() tea.Msg { return DeckCreatedMsg{Deck: d} }
			}
			d := m.decks[m.cursor-1]
			return m, func() tea.Msg { return DeckSelectedMsg{Deck: &d} }
		case key.Matches(msg, dbKeys.Delete):
			if m.cursor > 0 {
				id := m.decks[m.cursor-1].ID
				return m, m.deleteDeckCmd(id)
			}
		case key.Matches(msg, dbKeys.Export):
			if m.cursor > 0 {
				d := m.decks[m.cursor-1]
				return m, exportDeckCmd(d, m.cardNames)
			}
		case key.Matches(msg, dbKeys.Import):
			p := NewImportPalette()
			if len(p.files) == 0 {
				m.status = "No .txt files found in current directory"
				m.statusErr = true
			} else {
				m.importPalette = p
				m.importOpen = true
			}
		}
	}
	return m, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("error loading decks: %v", m.err)
	}
	if !m.loaded {
		return "Loading decks..."
	}

	if m.importOpen {
		return renderImportOverlay(m.importPalette.View(), m.width, m.height)
	}

	listW := m.width * 2 / 5
	detailPad := 2
	detailW := m.width - listW - detailPad

	list := m.renderList(listW)
	detail := m.renderDetail(detailW)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(listW).Height(m.height).Render(list),
		lipgloss.NewStyle().Width(m.width-listW).Height(m.height).PaddingLeft(detailPad).Render(detail),
	)
}

func renderImportOverlay(palette string, w, h int) string {
	y := max((h-lipgloss.Height(palette))/4, 0)
	return lipgloss.Place(w, h,
		lipgloss.Center, lipgloss.Top,
		lipgloss.NewStyle().MarginTop(y).Render(palette),
		lipgloss.WithWhitespaceChars(" "),
	)
}

func (m Model) renderList(w int) string {
	var sb strings.Builder

	rightW := 9 // "(50) ✓  " fits in 9
	nameW := w - rightW
	if nameW < 4 {
		nameW = 4
	}

	divider := styles.StyleDetailDivider.Render(strings.Repeat("─", w))

	sb.WriteString(styles.StyleDetailTitle.Render("Decks"))
	sb.WriteString("\n")
	sb.WriteString(divider)
	sb.WriteString("\n")

	// "New Deck" row
	newRow := lipgloss.NewStyle().Width(w).Render("★ New Deck")
	if m.cursor == 0 {
		sb.WriteString(styles.StyleRowSelected.Render(newRow))
	} else {
		sb.WriteString(styles.StyleRowNormal.Render(newRow))
	}
	sb.WriteString("\n")

	if len(m.decks) > 0 {
		sb.WriteString(divider)
		sb.WriteString("\n")
	}

	for i, d := range m.decks {
		total := d.TotalCards()
		var badge string
		if total == models.DeckMaxSize {
			badge = fmt.Sprintf("(%d)✓", total)
		} else {
			badge = fmt.Sprintf("(%d)", total)
		}

		name := lipgloss.NewStyle().Width(nameW).Render(truncate(d.Name, nameW-1))
		right := lipgloss.NewStyle().Width(rightW).Align(lipgloss.Right).Render(badge)
		row := name + right

		if m.cursor == i+1 {
			sb.WriteString(styles.StyleRowSelected.Render(row))
		} else {
			sb.WriteString(styles.StyleRowNormal.Render(row))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(divider)
	sb.WriteString("\n")
	sb.WriteString(styles.StylePaletteHint.Render("↑↓ · enter activate · n new · x delete"))
	sb.WriteString("\n")
	sb.WriteString(styles.StylePaletteHint.Render("e export · i import"))

	if m.status != "" {
		sb.WriteString("\n")
		var statusStyle lipgloss.Style
		if m.statusErr {
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Italic(true)
		} else {
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Italic(true)
		}
		sb.WriteString(statusStyle.Render(m.status))
	}

	return sb.String()
}

func (m Model) renderDetail(w int) string {
	var sb strings.Builder

	var deck *models.Deck
	if m.cursor > 0 && m.cursor-1 < len(m.decks) {
		d := m.decks[m.cursor-1]
		deck = &d
	}

	divider := styles.StyleDetailDivider.Render(strings.Repeat("─", w))

	if deck == nil {
		sb.WriteString(styles.StyleDetailTitle.Render("★ New Deck"))
		sb.WriteString("\n")
		sb.WriteString(divider)
		sb.WriteString("\n")
		sb.WriteString(styles.StyleDetailLabel.Render("Press enter or n to create a new deck,"))
		sb.WriteString("\n")
		sb.WriteString(styles.StyleDetailLabel.Render("then add cards from the Card Library."))
		return sb.String()
	}

	// Title row: name left, progress right
	total := deck.TotalCards()
	progressStr := fmt.Sprintf("%d / %d", total, models.DeckMaxSize)
	progressW := len(progressStr)
	nameColW := w - progressW
	if nameColW < 1 {
		nameColW = 1
	}

	var progressStyle lipgloss.Style
	if total == models.DeckMaxSize {
		progressStyle = styles.StyleDetailTitle
	} else {
		progressStyle = styles.StyleDetailLabel
	}

	titleName := styles.StyleDetailTitle.Width(nameColW).Render(truncate(deck.Name, nameColW-1))
	titleCount := progressStyle.Width(progressW).Align(lipgloss.Right).Render(progressStr)
	sb.WriteString(titleName + titleCount)
	sb.WriteString("\n")
	sb.WriteString(divider)
	sb.WriteString("\n")

	if len(deck.Entries) == 0 {
		sb.WriteString(styles.StyleDetailLabel.Render("(empty)"))
		sb.WriteString("\n")
		sb.WriteString(styles.StyleDetailLabel.Render("Activate this deck, then press Enter on cards in the library."))
	} else {
		// Column layout: Code | Name | ×N
		codeW := 10
		qtyW := 4
		nameW := w - codeW - qtyW
		if nameW < 1 {
			nameW = 1
		}

		// Column headers
		hCode := styles.StyleColumnHeader.Width(codeW).Render("Code")
		hName := styles.StyleColumnHeader.Width(nameW).Render("Name")
		hQty := styles.StyleColumnHeader.Width(qtyW).Align(lipgloss.Right).Render("Qty")
		sb.WriteString(hCode + hName + hQty)
		sb.WriteString("\n")
		sb.WriteString(styles.StyleDetailDivider.Render(strings.Repeat("─", w)))
		sb.WriteString("\n")

		for _, e := range deck.Entries {
			cardName := m.cardNames[e.CardCode]
			code := styles.StyleDetailLabel.Width(codeW).Render(e.CardCode)
			name := lipgloss.NewStyle().Width(nameW).Render(truncate(cardName, nameW-1))
			qty := styles.StyleDetailValue.Width(qtyW).Align(lipgloss.Right).Render(fmt.Sprintf("×%d", e.Quantity))
			sb.WriteString(styles.StyleRowNormal.Render(code + name + qty))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(divider)
	sb.WriteString("\n")
	sb.WriteString(styles.StylePaletteHint.Render("enter: activate · x: delete · e: export · i: import"))

	return sb.String()
}

// ── Commands ──────────────────────────────────────────────────────────────────

func (m Model) loadDecks() tea.Cmd {
	s := m.store
	return func() tea.Msg {
		decks, err := s.ListDecks()
		if err != nil {
			return decksLoadedMsg{err: err}
		}

		// Collect all unique card codes across every deck entry.
		seen := map[string]struct{}{}
		for _, d := range decks {
			for _, e := range d.Entries {
				seen[e.CardCode] = struct{}{}
			}
		}
		codes := make([]string, 0, len(seen))
		for c := range seen {
			codes = append(codes, c)
		}

		names, err := s.GetCardNamesByCodes(codes)
		if err != nil {
			names = map[string]string{}
		}

		return decksLoadedMsg{decks: decks, cardNames: names}
	}
}

func (m Model) deleteDeckCmd(id string) tea.Cmd {
	s := m.store
	return func() tea.Msg {
		_ = s.DeleteDeck(id)
		return DeckDeletedMsg{ID: id}
	}
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
