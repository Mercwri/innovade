package library

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Mercwri/innovade/internal/models"
	"github.com/Mercwri/innovade/internal/ui/styles"
)

func renderDetail(card *models.Card, width int) string {
	if card == nil {
		return styles.StyleDetailPanel.Width(width).Render(
			styles.StyleDetailLabel.Render("No card selected"),
		)
	}

	inner := width - 2 // account for panel padding

	var sb strings.Builder

	// Line 1: name · code · rarity · color swatch
	name := styles.StyleDetailTitle.Render(card.Name)
	code := styles.StyleDetailLabel.Render(fmt.Sprintf("[%s]", card.CardCode))
	rarity := styles.RarityStyle(string(card.Rarity)).Render(string(card.Rarity))
	colorDot := ""
	if len(card.Colors) > 0 {
		colorDot = styles.CardColorSwatch(string(card.Colors[0])) + " " +
			styles.StyleDetailValue.Render(string(card.Colors[0]))
	}
	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
		name, "  ", code, "  ", rarity, "  ", colorDot,
	))
	sb.WriteString("\n")

	// Line 2: stats
	sb.WriteString(renderStatLine(card))
	sb.WriteString("\n")

	// Line 3: types
	if len(card.Types) > 0 {
		sb.WriteString(styles.StyleDetailLabel.Render(strings.Join(card.Types, ", ")))
		sb.WriteString("\n")
	}

	// Line 4: locations
	if len(card.Locations) > 0 {
		locs := make([]string, len(card.Locations))
		for i, l := range card.Locations {
			locs[i] = string(l)
		}
		sb.WriteString(styles.StyleDetailLabel.Render("⬡ " + strings.Join(locs, " · ")))
		sb.WriteString("\n")
	}

	// Divider
	sb.WriteString(styles.StyleDetailDivider.Render(strings.Repeat("─", inner)))
	sb.WriteString("\n")

	// Effect text
	if card.Effect != "" {
		wrapped := wordWrap(card.Effect, inner)
		sb.WriteString(styles.StyleDetailEffect.Render(wrapped))
		sb.WriteString("\n")
	}

	// Burst
	if card.HasBurst() {
		sb.WriteString("\n")
		sb.WriteString(styles.StyleDetailLabel.Render("【Burst】"))
		sb.WriteString("\n")
		sb.WriteString(styles.StyleDetailBurst.Render(wordWrap(card.Burst, inner)))
		sb.WriteString("\n")
	}

	// Link requirement
	if card.HasLinkRequirement() {
		sb.WriteString("\n")
		sb.WriteString(styles.StyleDetailLabel.Render("Link: "))
		sb.WriteString(styles.StyleDetailValue.Render(card.LinkRequirement))
		sb.WriteString("\n")
	}

	// Alt art indicator
	if card.AltArtCount() > 0 {
		sb.WriteString(styles.StyleDetailLabel.Render(
			fmt.Sprintf("  +%d alternate art(s)", card.AltArtCount()),
		))
		sb.WriteString("\n")
	}

	return styles.StyleDetailPanel.Width(width).Render(sb.String())
}

func renderStatLine(card *models.Card) string {
	var parts []string

	switch card.Category {
	case models.CategoryUnit:
		parts = append(parts,
			statPair("Lv", card.Level),
			statPair("Cost", card.Cost),
			statPair("AP", card.AP),
			statPair("HP", card.HP),
		)
	case models.CategoryBase:
		parts = append(parts,
			statPair("Lv", card.Level),
			statPair("Cost", card.Cost),
			statPair("HP", card.HP),
		)
	case models.CategoryCommand, models.CategoryPilot:
		parts = append(parts,
			statPair("Lv", card.Level),
			statPair("Cost", card.Cost),
		)
	case models.CategoryUnitToken:
		parts = append(parts,
			statPair("AP", card.AP),
			statPair("HP", card.HP),
		)
	}

	return styles.StyleDetailValue.Render(strings.Join(parts, " · "))
}

func statPair(label string, val int) string {
	return fmt.Sprintf("%s %d",
		styles.StyleDetailLabel.Render(label),
		val,
	)
}

// wordWrap breaks s into lines of at most maxWidth runes, breaking on spaces.
func wordWrap(s string, maxWidth int) string {
	runes := []rune(s)
	if maxWidth <= 0 || len(runes) <= maxWidth {
		return s
	}

	var lines []string
	for len(runes) > maxWidth {
		cut := maxWidth
		for cut > 0 && runes[cut] != ' ' {
			cut--
		}
		if cut == 0 {
			cut = maxWidth // no space found, hard break
		}
		lines = append(lines, string(runes[:cut]))
		runes = runes[cut:]
		for len(runes) > 0 && runes[0] == ' ' {
			runes = runes[1:]
		}
	}
	if len(runes) > 0 {
		lines = append(lines, string(runes))
	}
	return strings.Join(lines, "\n")
}
