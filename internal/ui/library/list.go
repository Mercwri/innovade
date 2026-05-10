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
)

func renderList(m Model) string {
	var sb strings.Builder

	// Status bar: count + active filter summary
	filterSummary := buildFilterSummary(m.filter)
	countStr := fmt.Sprintf("%d cards", m.total)
	if filterSummary != "" {
		countStr += "  · " + filterSummary
	}
	sb.WriteString(styles.StyleDetailLabel.Render(countStr))
	sb.WriteString("\n")

	// Column headers
	sb.WriteString(renderColumnHeaders())
	sb.WriteString("\n")
	sb.WriteString(styles.StyleDetailDivider.Render(strings.Repeat("─", m.width)))
	sb.WriteString("\n")

	// Visible rows
	end := m.offset + m.listH - 3 // account for header rows
	if end > len(m.cards) {
		end = len(m.cards)
	}

	for i := m.offset; i < end; i++ {
		card := m.cards[i]
		row := renderRow(card, i == m.cursor)
		sb.WriteString(row)
		sb.WriteString("\n")
	}

	// Scroll hint
	if len(m.cards) > m.listH {
		pct := 0
		if len(m.cards) > 0 {
			pct = (m.cursor + 1) * 100 / len(m.cards)
		}
		hint := fmt.Sprintf("  %d%%  ↑↓/jk to navigate · g/G top/bottom", pct)
		sb.WriteString(styles.StylePaletteHint.Render(hint))
	}

	return sb.String()
}

func renderColumnHeaders() string {
	id := styles.StyleColumnHeader.Width(colIDWidth).Render("ID")
	name := styles.StyleColumnHeader.Width(colNameWidth).Render("Name")
	lvCost := styles.StyleColumnHeader.Width(colLvCostWidth).Render("Lv/Cost")
	apHP := styles.StyleColumnHeader.Width(colAPHPWidth).Render("AP/HP")
	color := styles.StyleColumnHeader.Width(colColorWidth).Render("Color")
	return lipgloss.JoinHorizontal(lipgloss.Top, id, name, lvCost, apHP, color)
}

func renderRow(card models.Card, selected bool) string {
	rowStyle := styles.StyleRowNormal
	if selected {
		rowStyle = styles.StyleRowSelected
	}

	id := lipgloss.NewStyle().Width(colIDWidth).Render(card.CardCode)
	name := lipgloss.NewStyle().Width(colNameWidth).Render(truncate(card.Name, colNameWidth-1))
	lvCost := lipgloss.NewStyle().Width(colLvCostWidth).Render(formatLvCost(card))
	apHP := lipgloss.NewStyle().Width(colAPHPWidth).Render(formatAPHP(card))
	colorCol := lipgloss.NewStyle().Width(colColorWidth).Render(formatColor(card))

	row := lipgloss.JoinHorizontal(lipgloss.Top, id, name, lvCost, apHP, colorCol)
	return rowStyle.Render(row)
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

func renderWithFilterPalette(list, detail, palette string, w, h int) string {
	base := renderLayout(list, detail, w, h)
	paletteW := lipgloss.Width(palette)
	x := (w - paletteW) / 2
	if x < 0 {
		x = 0
	}
	return lipgloss.Place(w, h,
		lipgloss.Left, lipgloss.Top,
		lipgloss.NewStyle().MarginLeft(x).MarginTop(3).Render(palette),
		lipgloss.WithWhitespaceChars(" "),
	)
	_ = base
	return base
}

func buildFilterSummary(f models.CardFilter) string {
	var parts []string
	if f.Name != "" {
		parts = append(parts, "name:"+f.Name)
	}
	for _, c := range f.Colors {
		parts = append(parts, string(c))
	}
	for _, cat := range f.Categories {
		parts = append(parts, string(cat))
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
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
