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
		countStr += "  · " + styles.StylePaletteHint.Render("filters paused")
	} else if summary := buildFilterSummary(m.filter); summary != "" {
		countStr += "  · " + summary
	}
	sb.WriteString(styles.StyleDetailLabel.Render(countStr))
	sb.WriteString("\n")

	// Column headers
	sb.WriteString(renderColumnHeaders(m.activeDeck != nil))
	sb.WriteString("\n")
	sb.WriteString(styles.StyleDetailDivider.Render(strings.Repeat("─", m.width)))
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
	var hint string
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
		hint += " · enter add · del remove"
	}
	sb.WriteString(styles.StylePaletteHint.Render(hint))

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

func renderLayout(list, detail string, w, h int) string {
	listH := h * 2 / 3
	detailH := h - listH

	listBox := lipgloss.NewStyle().
		Height(listH).
		Width(w).
		Render(list)

	detailBox := lipgloss.NewStyle().
		Height(detailH).
		Width(w).
		Render(detail)

	return lipgloss.JoinVertical(lipgloss.Left, listBox, detailBox)
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
	)
}

func renderWithFilterPalette(palette string, w, h int) string {
	y := max((h-lipgloss.Height(palette))/4, 0)
	return lipgloss.Place(w, h,
		lipgloss.Center, lipgloss.Top,
		lipgloss.NewStyle().MarginTop(y).Render(palette),
		lipgloss.WithWhitespaceChars(" "),
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
	if f.SetCode != "" {
		parts = append(parts, f.SetCode)
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
