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
	Up         key.Binding
	Down       key.Binding
	Select     key.Binding
	New        key.Binding
	Delete     key.Binding
	Export     key.Binding
	Import     key.Binding
	Rename     key.Binding
	ScrollUp   key.Binding
	ScrollDown key.Binding
}{
	Up:         key.NewBinding(key.WithKeys("up", "k")),
	Down:       key.NewBinding(key.WithKeys("down", "j")),
	Select:     key.NewBinding(key.WithKeys("enter")),
	New:        key.NewBinding(key.WithKeys("n")),
	Delete:     key.NewBinding(key.WithKeys("x")),
	Export:     key.NewBinding(key.WithKeys("e")),
	Import:     key.NewBinding(key.WithKeys("i")),
	Rename:     key.NewBinding(key.WithKeys("r")),
	ScrollUp:   key.NewBinding(key.WithKeys("pgup", "ctrl+u")),
	ScrollDown: key.NewBinding(key.WithKeys("pgdown", "ctrl+d")),
}

// DeckSelectedMsg is sent when the user activates a deck for library editing.
type DeckSelectedMsg struct{ Deck *models.Deck }

// DeckCreatedMsg is sent when a new deck is requested.
type DeckCreatedMsg struct{ Deck *models.Deck }

// DeckDeletedMsg is sent after a deck is removed from the store.
type DeckDeletedMsg struct{ ID string }

type decksLoadedMsg struct {
	decks        []models.Deck
	cardStats    map[string]store.CardStat
	activeDeckID string // from session; "" if none
	err          error
}

type renameResultMsg struct {
	id   string
	name string
	err  error
}

// Model is the deck builder Bubble Tea model.
type Model struct {
	store     *store.Store
	decks     []models.Deck
	cardStats map[string]store.CardStat

	cursor     int // 0 = ★ New Deck, 1+ = deck index+1
	listOffset int // scroll offset for left-pane deck list

	entryOffset int // scroll offset for right-pane entry list

	width  int
	height int
	loaded bool
	err    error

	status    string
	statusErr bool

	renaming    bool
	renameInput string

	importOpen    bool
	importPalette ImportPalette

	urlImportOpen bool
	urlInput      string

	exportOpen    bool
	exportPalette ExportPalette
}

func New(s *store.Store) Model {
	return Model{store: s, cardStats: map[string]store.CardStat{}}
}

func (m Model) Init() tea.Cmd { return m.loadDecks(true) }

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m Model) Reload() tea.Cmd { return m.loadDecks(false) }

// CardStats exposes the cached card stats for use by the root app.
func (m Model) CardStats() map[string]store.CardStat { return m.cardStats }

// Capturing reports whether the model is consuming all key input (text fields,
// overlays). app.go uses this to bypass global hotkeys.
func (m Model) Capturing() bool {
	return m.urlImportOpen || m.renaming || m.importOpen || m.exportOpen
}

func (m Model) selectedDeck() *models.Deck {
	if m.cursor > 0 && m.cursor-1 < len(m.decks) {
		d := m.decks[m.cursor-1]
		return &d
	}
	return nil
}

func (m Model) cardName(code string) string {
	if s, ok := m.cardStats[code]; ok {
		return s.Name
	}
	return code
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case decksLoadedMsg:
		m.loaded = true
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.decks = msg.decks
		if msg.cardStats != nil {
			m.cardStats = msg.cardStats
		}
		if m.cursor > len(m.decks) {
			m.cursor = len(m.decks)
		}
		// Restore previously active deck from session.
		if msg.activeDeckID != "" {
			for i, d := range m.decks {
				if d.ID == msg.activeDeckID {
					m.cursor = i + 1 // +1: cursor 0 = ★ New Deck
					deck := m.decks[i]
					return m, func() tea.Msg { return DeckSelectedMsg{Deck: &deck} }
				}
			}
		}
		return m, nil

	case renameResultMsg:
		m.renaming = false
		if msg.err != nil {
			m.status = "rename failed: " + msg.err.Error()
			m.statusErr = true
		} else {
			for i := range m.decks {
				if m.decks[i].ID == msg.id {
					m.decks[i].Name = msg.name
					break
				}
			}
			m.status = "Renamed → " + msg.name
			m.statusErr = false
		}
		return m, nil

	case exportResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("export failed: %v", msg.err)
			m.statusErr = true
		} else if msg.path != "" {
			m.status = fmt.Sprintf("Exported → %s", msg.path)
			m.statusErr = false
		} else {
			m.status = msg.note
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
			return m, m.loadDecks(false)
		}
		return m, nil

	case tea.KeyMsg:
		// Rename mode captures everything.
		if m.renaming {
			switch msg.Type {
			case tea.KeyEnter:
				if name := strings.TrimSpace(m.renameInput); name != "" {
					id := m.decks[m.cursor-1].ID
					return m, m.renameDeckCmd(id, name)
				}
				m.renaming = false
			case tea.KeyEscape:
				m.renaming = false
				m.renameInput = ""
			case tea.KeyBackspace, tea.KeyDelete:
				runes := []rune(m.renameInput)
				if len(runes) > 0 {
					m.renameInput = string(runes[:len(runes)-1])
				}
			case tea.KeyRunes:
				m.renameInput += string(msg.Runes)
			}
			return m, nil
		}

		// Export palette overlay.
		if m.exportOpen {
			var action exportAction
			m.exportPalette, action = m.exportPalette.Update(msg)
			switch action {
			case exportActionClose:
				m.exportOpen = false
			case exportActionFile:
				m.exportOpen = false
				d := m.decks[m.cursor-1]
				stats := m.cardStats
				nameFor := func(code string) string {
					if s, ok := stats[code]; ok {
						return s.Name
					}
					return code
				}
				return m, exportDeckCmd(d, nameFor)
			case exportActionClipboard:
				m.exportOpen = false
				d := m.decks[m.cursor-1]
				stats := m.cardStats
				nameFor := func(code string) string {
					if s, ok := stats[code]; ok {
						return s.Name
					}
					return code
				}
				return m, copyToClipboardCmd(formatDeckList(&d, nameFor), "Decklist copied to clipboard")
			case exportActionMSALink:
				m.exportOpen = false
				d := m.decks[m.cursor-1]
				return m, copyToClipboardCmd(formatMSALink(&d), "MSA link copied to clipboard")
			}
			return m, nil
		}

		// URL import input mode.
		if m.urlImportOpen {
			switch msg.Type {
			case tea.KeyEnter:
				if rawURL := strings.TrimSpace(m.urlInput); rawURL != "" {
					m.urlImportOpen = false
					return m, importDeckUrlCmd(m.store, rawURL)
				}
				m.urlImportOpen = false
			case tea.KeyEscape:
				m.urlImportOpen = false
				m.urlInput = ""
			case tea.KeyBackspace, tea.KeyDelete:
				runes := []rune(m.urlInput)
				if len(runes) > 0 {
					m.urlInput = string(runes[:len(runes)-1])
				}
			case tea.KeyRunes:
				m.urlInput += string(msg.Runes)
			}
			return m, nil
		}

		// Import palette overlay.
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
			case importActionURL:
				m.importOpen = false
				m.urlImportOpen = true
				m.urlInput = ""
			}
			return m, nil
		}

		m.status = ""
		total := 1 + len(m.decks) // ★ New + saved decks

		switch {
		case key.Matches(msg, dbKeys.Up):
			if m.cursor > 0 {
				m.cursor--
				m.entryOffset = 0
				m.scrollListToSelected()
			}
		case key.Matches(msg, dbKeys.Down):
			if m.cursor < total-1 {
				m.cursor++
				m.entryOffset = 0
				m.scrollListToSelected()
			}
		case key.Matches(msg, dbKeys.ScrollUp):
			if m.entryOffset > 0 {
				m.entryOffset--
			}
		case key.Matches(msg, dbKeys.ScrollDown):
			if d := m.selectedDeck(); d != nil {
				max := len(d.Entries) - m.entryVisibleRows()
				if max > 0 && m.entryOffset < max {
					m.entryOffset++
				}
			}
		case key.Matches(msg, dbKeys.Rename):
			if m.cursor > 0 {
				m.renaming = true
				m.renameInput = m.decks[m.cursor-1].Name
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
				m.exportPalette = ExportPalette{}
				m.exportOpen = true
			}
		case key.Matches(msg, dbKeys.Import):
			m.importPalette = NewImportPalette()
			m.importOpen = true
		}
	}
	return m, nil
}

// ── Layout helpers ────────────────────────────────────────────────────────────

// listVisibleRows: rows available in the left pane for deck name entries.
// Fixed rows: title(1)+divider(1)+newDeck(1)+postNewDivider(1)+
//             analysisDivider(1)+curveHeader(1)+curveRows(4)+totalsDivider(1)+totalsRow(1)+
//             barRows(4)+barAxis(1)+
//             hintDivider(1)+hints(2)+statusReserve(1) = 21
func (m Model) listVisibleRows() int {
	v := m.height - 21
	if v < 0 {
		return 0
	}
	return v
}

// entryVisibleRows: rows available in the right pane for deck entries.
// Fixed rows: title(1)+divider(1)+colHeader(1)+colDivider(1)+hintDivider(1)+hints(1) = 6
func (m Model) entryVisibleRows() int {
	v := m.height - 6
	if v < 0 {
		return 0
	}
	return v
}

func (m *Model) scrollListToSelected() {
	if m.cursor == 0 {
		return // ★ New Deck is always visible above the scrollable area
	}
	deckIdx := m.cursor - 1
	vr := m.listVisibleRows()
	if deckIdx < m.listOffset {
		m.listOffset = deckIdx
	}
	if deckIdx >= m.listOffset+vr {
		m.listOffset = deckIdx - vr + 1
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("error loading decks: %v", m.err)
	}
	if !m.loaded {
		return "Loading decks..."
	}
	if m.urlImportOpen {
		return renderURLImportOverlay(m.urlInput, m.width, m.height)
	}
	if m.importOpen {
		return renderImportOverlay(m.importPalette.View(), m.width, m.height)
	}
	if m.exportOpen {
		return renderImportOverlay(m.exportPalette.View(), m.width, m.height)
	}

	leftW := m.width * 2 / 5
	rightW := m.width - leftW

	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(leftW).Height(m.height).Background(styles.BgBase).Render(m.renderLeft(leftW)),
		lipgloss.NewStyle().Width(rightW).Height(m.height).Background(styles.BgBase).Render(m.renderRight(rightW)),
	)
}

func renderImportOverlay(palette string, w, h int) string {
	y := max((h-lipgloss.Height(palette))/4, 0)
	return lipgloss.Place(w, h,
		lipgloss.Center, lipgloss.Top,
		lipgloss.NewStyle().MarginTop(y).Render(palette),
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceBackground(styles.BgBase),
	)
}

// ── Left pane ─────────────────────────────────────────────────────────────────

func (m Model) renderLeft(w int) string {
	var sb strings.Builder
	div := styles.StyleDetailDivider.Render(strings.Repeat("─", w))

	// Header
	sb.WriteString(styles.StyleDetailTitle.Render("Decks"))
	sb.WriteString("\n")
	sb.WriteString(div)
	sb.WriteString("\n")

	// ★ New Deck row (always visible)
	newRow := lipgloss.NewStyle().Width(w).Render("★ New Deck")
	if m.cursor == 0 {
		sb.WriteString(styles.StyleRowSelected.Render(newRow))
	} else {
		sb.WriteString(styles.StyleRowNormal.Render(newRow))
	}
	sb.WriteString("\n")
	sb.WriteString(div)
	sb.WriteString("\n")

	// Scrollable deck list
	vr := m.listVisibleRows()
	end := m.listOffset + vr
	if end > len(m.decks) {
		end = len(m.decks)
	}
	for i := m.listOffset; i < end; i++ {
		sb.WriteString(m.renderDeckRow(i, w))
		sb.WriteString("\n")
	}
	// Pad empty rows so analysis stays at a fixed position
	for i := end - m.listOffset; i < vr; i++ {
		sb.WriteString(lipgloss.NewStyle().Width(w).Render(""))
		sb.WriteString("\n")
	}

	// Analysis section
	sb.WriteString(div)
	sb.WriteString("\n")
	sb.WriteString(m.renderCurveGrid(w))

	// Hints
	sb.WriteString(div)
	sb.WriteString("\n")
	sb.WriteString(styles.StyleContentHint.Render("↑↓ nav · enter activate · n new · x del"))
	sb.WriteString("\n")
	sb.WriteString(styles.StyleContentHint.Render("r rename · e export · i import · PgDn/PgUp scroll →"))

	if m.status != "" {
		sb.WriteString("\n")
		var st lipgloss.Style
		if m.statusErr {
			st = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Italic(true)
		} else {
			st = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Italic(true)
		}
		sb.WriteString(st.Render(truncate(m.status, w-1)))
	}

	return sb.String()
}

func (m Model) renderDeckRow(deckIdx int, w int) string {
	d := m.decks[deckIdx]
	selected := m.cursor == deckIdx+1

	const badgeW = 9
	nameW := w - badgeW
	if nameW < 4 {
		nameW = 4
	}

	total := d.TotalCards()
	var badge string
	if total >= models.DeckMaxSize {
		badge = fmt.Sprintf("(%d)✓", total)
	} else {
		badge = fmt.Sprintf("(%d)", total)
	}

	var nameStr string
	if selected && m.renaming {
		// Show editable input with block cursor
		input := m.renameInput + "█"
		nameStr = lipgloss.NewStyle().Width(nameW).Render(truncate(input, nameW-1))
	} else {
		nameStr = lipgloss.NewStyle().Width(nameW).Render(truncate(d.Name, nameW-1))
	}
	right := lipgloss.NewStyle().Width(badgeW).Align(lipgloss.Right).Render(badge)
	row := nameStr + right

	if selected {
		return styles.StyleRowSelected.Render(row)
	}
	return styles.StyleRowNormal.Render(row)
}

// renderCurveGrid draws the level×category grid, a totals row, and a bar chart.
func (m Model) renderCurveGrid(w int) string {
	const (
		minLv  = 1
		maxLv  = 9
		catW   = 4 // label column width
		colW   = 3 // per-level and Σ column width
		chartH = 4 // bar chart height in rows
	)

	type catDef struct {
		label string
		kind  models.Category
	}
	cats := []catDef{
		{"Unt", models.CategoryUnit},
		{"Cmd", models.CategoryCommand},
		{"Plt", models.CategoryPilot},
		{"Bas", models.CategoryBase},
	}

	deck := m.selectedDeck()

	// counts[cat][level], totals[level]
	counts := [4][maxLv + 1]int{}
	var lvTotals [maxLv + 1]int
	if deck != nil {
		for _, e := range deck.Entries {
			s, ok := m.cardStats[e.CardCode]
			if !ok {
				continue
			}
			lv := s.Level
			if lv < 0 {
				lv = 0
			}
			if lv > maxLv {
				lv = maxLv
			}
			lvTotals[lv] += e.Quantity
			for ci, cat := range cats {
				if s.Category == cat.kind {
					counts[ci][lv] += e.Quantity
				}
			}
		}
	}

	var sb strings.Builder

	// ── Header row ────────────────────────────────────────────────────────────
	hdr := fmt.Sprintf("%-*s", catW, "")
	for lv := minLv; lv <= maxLv; lv++ {
		hdr += fmt.Sprintf("%*d", colW, lv)
	}
	hdr += fmt.Sprintf("%*s", colW, "Σ")
	sb.WriteString(styles.StyleColumnHeader.Render(hdr))
	sb.WriteString("\n")

	// ── Category rows ─────────────────────────────────────────────────────────
	for ci, cat := range cats {
		total := 0
		row := fmt.Sprintf("%-*s", catW, cat.label)
		for lv := minLv; lv <= maxLv; lv++ {
			n := counts[ci][lv]
			total += n
			if n == 0 {
				row += fmt.Sprintf("%*s", colW, "·")
			} else {
				row += fmt.Sprintf("%*d", colW, n)
			}
		}
		if total == 0 {
			row += fmt.Sprintf("%*s", colW, "·")
		} else {
			row += fmt.Sprintf("%*d", colW, total)
		}
		sb.WriteString(styles.StyleDetailLabel.Render(row))
		sb.WriteString("\n")
	}

	// ── Totals row ────────────────────────────────────────────────────────────
	divLen := catW + (maxLv-minLv+1)*colW + colW
	sb.WriteString(styles.StyleDetailDivider.Render(strings.Repeat("─", divLen)))
	sb.WriteString("\n")

	grandTotal := 0
	totRow := fmt.Sprintf("%-*s", catW, "Tot")
	for lv := minLv; lv <= maxLv; lv++ {
		n := lvTotals[lv]
		grandTotal += n
		if n == 0 {
			totRow += fmt.Sprintf("%*s", colW, "·")
		} else {
			totRow += fmt.Sprintf("%*d", colW, n)
		}
	}
	if grandTotal == 0 {
		totRow += fmt.Sprintf("%*s", colW, "·")
	} else {
		totRow += fmt.Sprintf("%*d", colW, grandTotal)
	}
	sb.WriteString(styles.StyleDetailValue.Render(totRow))
	sb.WriteString("\n")

	// ── Bar chart (level distribution) ───────────────────────────────────────
	maxV := 0
	for lv := minLv; lv <= maxLv; lv++ {
		if lvTotals[lv] > maxV {
			maxV = lvTotals[lv]
		}
	}

	for row := chartH; row >= 1; row-- {
		line := fmt.Sprintf("%-*s", catW, "")
		for lv := minLv; lv <= maxLv; lv++ {
			var cell string
			if maxV > 0 && lvTotals[lv]*chartH/maxV >= row {
				cell = "█"
			} else {
				cell = " "
			}
			line += fmt.Sprintf("%*s", colW, cell)
		}
		sb.WriteString(styles.StyleDetailValue.Render(line))
		sb.WriteString("\n")
	}
	// Axis label row
	axis := fmt.Sprintf("%-*s", catW, "")
	for lv := minLv; lv <= maxLv; lv++ {
		axis += fmt.Sprintf("%*d", colW, lv)
	}
	sb.WriteString(styles.StyleDetailLabel.Render(axis))
	sb.WriteString("\n")

	return sb.String()
}

// ── Right pane ────────────────────────────────────────────────────────────────

func (m Model) renderRight(w int) string {
	var sb strings.Builder
	div := styles.StyleDetailDivider.Render(strings.Repeat("─", w))

	deck := m.selectedDeck()

	if deck == nil {
		sb.WriteString(styles.StyleDetailTitle.Render("★ New Deck"))
		sb.WriteString("\n")
		sb.WriteString(div)
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
	nameColW := w - progressW - 1
	if nameColW < 1 {
		nameColW = 1
	}

	var progressStyle lipgloss.Style
	if total >= models.DeckMaxSize {
		progressStyle = styles.StyleDetailTitle
	} else {
		progressStyle = styles.StyleDetailLabel
	}
	sb.WriteString(styles.StyleDetailTitle.Width(nameColW).Render(truncate(deck.Name, nameColW-1)))
	sb.WriteString(progressStyle.Width(progressW).Align(lipgloss.Right).Render(progressStr))
	sb.WriteString("\n")
	sb.WriteString(div)
	sb.WriteString("\n")

	if len(deck.Entries) == 0 {
		sb.WriteString(styles.StyleDetailLabel.Render("(empty)  Activate this deck, then press Enter on cards."))
		sb.WriteString("\n")
	} else {
		// Column headers
		const codeW, qtyW = 10, 4
		nameW := w - codeW - qtyW
		if nameW < 1 {
			nameW = 1
		}
		sb.WriteString(styles.StyleColumnHeader.Width(codeW).Render("Code"))
		sb.WriteString(styles.StyleColumnHeader.Width(nameW).Render("Name"))
		sb.WriteString(styles.StyleColumnHeader.Width(qtyW).Align(lipgloss.Right).Render("Qty"))
		sb.WriteString("\n")
		sb.WriteString(div)
		sb.WriteString("\n")

		// Scrollable entry list
		vr := m.entryVisibleRows()
		start := m.entryOffset
		if start > len(deck.Entries) {
			start = len(deck.Entries)
		}
		end := start + vr
		if end > len(deck.Entries) {
			end = len(deck.Entries)
		}

		for _, e := range deck.Entries[start:end] {
			code := styles.StyleDetailLabel.Width(codeW).Render(e.CardCode)
			name := lipgloss.NewStyle().Width(nameW).Render(truncate(m.cardName(e.CardCode), nameW-1))
			qty := styles.StyleDetailValue.Width(qtyW).Align(lipgloss.Right).Render(fmt.Sprintf("×%d", e.Quantity))
			sb.WriteString(styles.StyleRowNormal.Render(code + name + qty))
			sb.WriteString("\n")
		}

		// Scroll hint
		if len(deck.Entries) > vr {
			pct := 0
			if len(deck.Entries) > 0 {
				pct = (start + 1) * 100 / len(deck.Entries)
			}
			sb.WriteString(div)
			sb.WriteString("\n")
			sb.WriteString(styles.StyleContentHint.Render(
				fmt.Sprintf("  %d%%  PgUp/PgDn to scroll deck", pct),
			))
		}
	}

	return sb.String()
}

// ── Commands ──────────────────────────────────────────────────────────────────

func (m Model) loadDecks(restoreSession bool) tea.Cmd {
	s := m.store
	return func() tea.Msg {
		decks, err := s.ListDecks()
		if err != nil {
			return decksLoadedMsg{err: err}
		}

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

		stats, err := s.GetCardStatsByCodes(codes)
		if err != nil {
			stats = map[string]store.CardStat{}
		}

		var activeDeckID string
		if restoreSession {
			activeDeckID, _ = s.GetSessionValue("active_deck_id")
		}
		return decksLoadedMsg{decks: decks, cardStats: stats, activeDeckID: activeDeckID}
	}
}

func (m Model) renameDeckCmd(id, name string) tea.Cmd {
	s := m.store
	return func() tea.Msg {
		err := s.RenameDeck(id, name)
		return renameResultMsg{id: id, name: name, err: err}
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
