package library

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Mercwri/innovade/internal/models"
	"github.com/Mercwri/innovade/internal/ui/styles"
)

// Column widths (characters)
const (
	colIDWidth     = 9
	colNameWidth   = 28
	colLvCostWidth = 7
	colAPHPWidth   = 7
	colColorWidth  = 8
	colDeckWidth   = 5
)

func renderList(m Model) string {
	var sb strings.Builder

	// Status bar: count + active filter summary
	countStr := fmt.Sprintf("%d cards", m.total)
	if !m.filtersActive {
		countStr += "  · " + styles.StyleContentHint.Render("filters paused")
	} else if summary := buildFilterSummary(m.filter); summary != "" {
		countStr += "  · " + summary
	}
	sb.WriteString(styles.StyleDetailLabel.Render(countStr))
	sb.WriteString("\n")

	// Column headers
	sb.WriteString(renderColumnHeaders(m.activeDeck != nil))
	sb.WriteString("\n")
	// listW-2: matches the box's content width in renderLayout (2 columns
	// are always reserved for the border), otherwise this full-width line
	// is wider than the box and lipgloss wraps it onto an extra row.
	sb.WriteString(styles.StyleDetailDivider.Render(strings.Repeat("─", m.listW-2)))
	sb.WriteString("\n")

	// Visible rows — visibleRows() already reserves the hint row so the
	// total line count (3 headers + cards + 1 hint) never exceeds m.listH.
	vr := m.visibleRows()
	end := m.offset + vr
	if end > len(m.cards) {
		end = len(m.cards)
	}

	for i := m.offset; i < end; i++ {
		card := m.cards[i]
		row := renderRow(card, i == m.cursor, m.activeDeck)
		sb.WriteString(row)
		sb.WriteString("\n")
	}

	// Help/scroll hint — always rendered (space is reserved by visibleRows).
	// Hints change based on panel focus
	var hint string
	if m.panelFocus == FocusDeck && m.activeDeck != nil {
		// Deck panel has focus
		hint = "↑↓/jk · enter remove · del remove · → back to library · ← focus library"
	} else {
		// Library panel has focus (default)
		if len(m.cards) > vr {
			pct := 0
			if len(m.cards) > 0 {
				pct = (m.cursor + 1) * 100 / len(m.cards)
			}
			hint = fmt.Sprintf("%d%% ↑↓/jk · g/G top/btm", pct)
		} else {
			hint = "↑↓/jk navigate · g/G top/bottom"
		}
		hint += " · t search · f filter · c clear · l links"
		if m.activeDeck != nil {
			hint += " · enter add · del remove · → deck"
		}
	}
	// Truncate to the box's content width — an unbounded line here wraps
	// inside the bordered panel, adding a row nothing else budgets for and
	// pushing content (including the app header) off the top of the screen.
	sb.WriteString(styles.StyleContentHint.Render(truncate(hint, m.listW-2)))

	return sb.String()
}

func renderColumnHeaders(deckActive bool) string {
	id := styles.StyleColumnHeader.Width(colIDWidth).Render("ID")
	name := styles.StyleColumnHeader.Width(colNameWidth).Render("Name")
	lvCost := styles.StyleColumnHeader.Width(colLvCostWidth).Render("Lv/Cost")
	apHP := styles.StyleColumnHeader.Width(colAPHPWidth).Render("AP/HP")
	color := styles.StyleColumnHeader.Width(colColorWidth).Render("Color")
	if deckActive {
		deck := styles.StyleColumnHeader.Width(colDeckWidth).Render("Deck")
		return lipgloss.JoinHorizontal(lipgloss.Top, id, name, lvCost, apHP, color, deck)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, id, name, lvCost, apHP, color)
}

func renderRow(card models.Card, selected bool, activeDeck *models.Deck) string {
	rowStyle := styles.StyleRowNormal
	if selected {
		rowStyle = styles.StyleRowSelected
	}

	id := lipgloss.NewStyle().Width(colIDWidth).Render(card.CardCode)
	name := lipgloss.NewStyle().Width(colNameWidth).Render(truncate(card.Name, colNameWidth-1))
	lvCost := lipgloss.NewStyle().Width(colLvCostWidth).Render(formatLvCost(card))
	apHP := lipgloss.NewStyle().Width(colAPHPWidth).Render(formatAPHP(card))
	colorCol := lipgloss.NewStyle().Width(colColorWidth).Render(formatColor(card))

	if activeDeck != nil {
		var deckStr string
		if n := activeDeck.CardCount(card.CardCode); n > 0 {
			deckStr = fmt.Sprintf("×%d", n)
		}
		deckCol := lipgloss.NewStyle().Width(colDeckWidth).Render(deckStr)
		return rowStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, id, name, lvCost, apHP, colorCol, deckCol))
	}
	return rowStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, id, name, lvCost, apHP, colorCol))
}

func formatLvCost(card models.Card) string {
	if card.Category == models.CategoryUnitToken {
		return "—"
	}
	return fmt.Sprintf("%d/%d", card.Level, card.Cost)
}

func formatAPHP(card models.Card) string {
	switch card.Category {
	case models.CategoryUnit, models.CategoryUnitToken:
		return fmt.Sprintf("%d/%d", card.AP, card.HP)
	case models.CategoryBase:
		return fmt.Sprintf("—/%d", card.HP)
	default:
		return "—"
	}
}

func formatColor(card models.Card) string {
	if len(card.Colors) == 0 {
		return styles.StyleRowDim.Render("—")
	}
	c := string(card.Colors[0])
	return styles.CardColorSwatch(c) + " " + c
}

func renderLayout(list, detail, deckPanel string, listW, w, h int, focus PanelFocus, deckActive bool) string {
	detailW := w - listW

	// Left panel: card library.
	// DoubleBorder adds 1 column on each side on top of Width(), so content
	// is always built 2 columns narrower than listW (see the divider in
	// renderList) — otherwise the left+right panes together overflow the
	// terminal width and every row wraps. The reservation is unconditional
	// so the panel doesn't change width when focus toggles the border.
	var listBox lipgloss.Style
	if focus == FocusLibrary {
		listBox = lipgloss.NewStyle().
			Background(styles.BgBase).
			BorderStyle(lipgloss.DoubleBorder()).
			Width(listW - 2)
	} else {
		listBox = lipgloss.NewStyle().
			Background(styles.BgBase).
			Width(listW - 2)
	}
	listRendered := listBox.Render(list)

	// Right panel
	if deckActive {
		// Two-panel layout (detail top, deck bottom)
		detailH := h * 55 / 100
		if detailH < 3 {
			detailH = 3
		}
		deckH := h - detailH
		if deckH < 3 {
			deckH = 3
			detailH = h - deckH
		}

		// Render with height constraints
		detailRendered := lipgloss.NewStyle().Width(detailW).Render(detail)
		
		var deckRendered string
		if focus == FocusDeck {
			// Same border-width compensation as the list panel above.
			deckRendered = lipgloss.NewStyle().
				BorderStyle(lipgloss.DoubleBorder()).
				Width(detailW - 2).
				Render(deckPanel)
		} else {
			deckRendered = lipgloss.NewStyle().Width(detailW).Render(deckPanel)
		}

		rightPanel := lipgloss.JoinVertical(lipgloss.Top, detailRendered, deckRendered)
		return lipgloss.JoinHorizontal(lipgloss.Top, listRendered, rightPanel)
	}

	// Single-panel layout (detail only)
	detailRendered := lipgloss.NewStyle().Width(detailW).Render(detail)
	return lipgloss.JoinHorizontal(lipgloss.Top, listRendered, detailRendered)
}

func renderDeckPanel(deck *models.Deck, width, height int, cursor int, offset int, cardNames map[string]string) string {
	if deck == nil {
		return styles.StyleDetailPanel.Width(width - 1).Height(height).Render(
			styles.StyleDetailLabel.Render("No deck selected"),
		)
	}

	var sb strings.Builder

	// Title
	sb.WriteString(styles.StyleDetailTitle.Render(truncate(deck.Name, width-3)))
	sb.WriteString("\n")
	sb.WriteString(styles.StyleDetailDivider.Render(strings.Repeat("─", width-3)))
	sb.WriteString("\n")

	if len(deck.Entries) == 0 {
		sb.WriteString(styles.StyleDetailLabel.Render("(empty)"))
		return sb.String()
	}

	// Column headers
	const codeW, qtyW = 9, 4
	nameW := width - codeW - qtyW - 2
	if nameW < 1 {
		nameW = 1
	}

	sb.WriteString(styles.StyleColumnHeader.Width(codeW).Render("Code"))
	sb.WriteString(styles.StyleColumnHeader.Width(nameW).Render("Name"))
	sb.WriteString(styles.StyleColumnHeader.Width(qtyW).Align(lipgloss.Right).Render("Qty"))
	sb.WriteString("\n")

	// Entries (limited to visible height)
	// Reserve space for title (1), divider (1), headers (1), totaling 3 lines
	vr := height - 3
	if vr < 1 {
		vr = 1
	}
	end := offset + vr
	if end > len(deck.Entries) {
		end = len(deck.Entries)
	}
	for i := offset; i < end; i++ {
		e := deck.Entries[i]
		rowStyle := styles.StyleRowNormal
		if i == cursor {
			rowStyle = styles.StyleRowSelected
		}
		displayName := e.CardCode
		if n, ok := cardNames[e.CardCode]; ok && n != "" {
			displayName = n
		}
		code := rowStyle.Width(codeW).Render(e.CardCode)
		name := rowStyle.Width(nameW).Render(truncate(displayName, nameW-1))
		qty := rowStyle.Width(qtyW).Align(lipgloss.Right).Render(fmt.Sprintf("×%d", e.Quantity))
		sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, code, name, qty))
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderWithTextInput(input string, w, h int) string {
	cursor := input + "█"
	body := "Search by name: " + cursor + "\n\n" +
		styles.StylePaletteHint.Render("↵ apply  ·  esc cancel")
	box := styles.StylePaletteOverlay.Width(44).Render(body)
	y := max((h-lipgloss.Height(box))/4, 0)
	return lipgloss.Place(w, h,
		lipgloss.Center, lipgloss.Top,
		lipgloss.NewStyle().MarginTop(y).Render(box),
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceBackground(styles.BgBase),
	)
}

func renderWithFilterPalette(palette string, w, h int) string {
	y := max((h-lipgloss.Height(palette))/4, 0)
	return lipgloss.Place(w, h,
		lipgloss.Center, lipgloss.Top,
		lipgloss.NewStyle().MarginTop(y).Render(palette),
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceBackground(styles.BgBase),
	)
}

func buildFilterSummary(f models.CardFilter) string {
	if f.Description != "" {
		return f.Description
	}
	var parts []string
	if f.Name != "" {
		parts = append(parts, "name:"+f.Name)
	}
	for _, c := range f.Colors {
		parts = append(parts, string(c))
	}
	if f.FindPilots {
		parts = append(parts, "Pilot")
	} else {
		for _, cat := range f.Categories {
			parts = append(parts, string(cat))
		}
	}
	if len(f.SetCodes) > 0 {
		parts = append(parts, strings.Join(f.SetCodes, ","))
	}
	if f.Type != "" {
		parts = append(parts, f.Type)
	}
	return strings.Join(parts, " · ")
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
