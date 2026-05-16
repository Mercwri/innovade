package library

import (
	"fmt"
	"strings"

	"github.com/Mercwri/innovade/internal/models"
	"github.com/Mercwri/innovade/internal/ui/styles"
)

func renderDetail(card *models.Card, width, height int, imagePath string) string {
	if card == nil {
		return styles.StyleDetailPanel.Width(width - 1).Render(
			styles.StyleDetailLabel.Render("No card selected"),
		)
	}

	// StyleDetailPanel has BorderLeft(1) + Padding(0,1).
	// Width() includes padding but not border, so Width(width-1) renders at total width `width`.
	// Content area = (width-1) - 2(padding) = width-3.
	inner := max(1, width-3)

	// Reserve ~45% of terminal height for card text; give the rest to the art.
	maxArtH := height * 55 / 100

	var sb strings.Builder

	// ── Art frame ────────────────────────────────────────────────────────────
	sb.WriteString(renderArtFrame(imagePath, inner, maxArtH))
	sb.WriteString("\n\n")

	// ── Name ─────────────────────────────────────────────────────────────────
	sb.WriteString(styles.StyleDetailTitle.Render(card.Name))
	sb.WriteString("\n")

	// ── Code · rarity · color ────────────────────────────────────────────────
	code := styles.StyleDetailLabel.Render("[" + card.CardCode + "]")
	rarity := styles.RarityStyle(string(card.Rarity)).Render(string(card.Rarity))
	infoLine := code + "  " + rarity
	if len(card.Colors) > 0 {
		colorPart := styles.CardColorSwatch(string(card.Colors[0])) + " " +
			styles.StyleDetailValue.Render(string(card.Colors[0]))
		infoLine += "  " + colorPart
	}
	sb.WriteString(infoLine)
	sb.WriteString("\n")

	// ── Stats ─────────────────────────────────────────────────────────────────
	if statsLine := renderStatLine(card); statsLine != "" {
		sb.WriteString(statsLine)
		sb.WriteString("\n")
	}

	// ── Types · locations ─────────────────────────────────────────────────────
	var tags []string
	if len(card.Types) > 0 {
		tags = append(tags, styles.StyleDetailLabel.Render(strings.Join(card.Types, ", ")))
	}
	if len(card.Locations) > 0 {
		locs := make([]string, len(card.Locations))
		for i, l := range card.Locations {
			locs[i] = string(l)
		}
		tags = append(tags, styles.StyleDetailLabel.Render("⬡ "+strings.Join(locs, " · ")))
	}
	if len(tags) > 0 {
		sb.WriteString(strings.Join(tags, "  "))
		sb.WriteString("\n")
	}

	// ── Divider ───────────────────────────────────────────────────────────────
	sb.WriteString(styles.StyleDetailDivider.Render(strings.Repeat("─", inner)))
	sb.WriteString("\n")

	// ── Effect ────────────────────────────────────────────────────────────────
	if card.Effect != "" {
		sb.WriteString("\n")
		sb.WriteString(styles.StyleDetailEffect.Render(wordWrap(card.Effect, inner)))
		sb.WriteString("\n")
	}

	// ── Burst ─────────────────────────────────────────────────────────────────
	if card.HasBurst() {
		sb.WriteString("\n")
		sb.WriteString(styles.StyleDetailBurst.Bold(true).Render("【Burst】"))
		sb.WriteString("\n")
		sb.WriteString(styles.StyleDetailBurst.Render(wordWrap(card.Burst, inner)))
		sb.WriteString("\n")
	}

	// ── Link requirement ──────────────────────────────────────────────────────
	if card.HasLinkRequirement() {
		sb.WriteString("\n")
		sb.WriteString(styles.StyleDetailLabel.Render("Link  "))
		sb.WriteString(styles.StyleDetailValue.Render(card.LinkRequirement))
		sb.WriteString("\n")
	}

	// ── Alt art count ─────────────────────────────────────────────────────────
	if card.AltArtCount() > 0 {
		sb.WriteString(styles.StyleContentHint.Render(
			fmt.Sprintf("+%d alternate art(s)", card.AltArtCount()),
		))
		sb.WriteString("\n")
	}

	return styles.StyleDetailPanel.Width(width - 1).Render(sb.String())
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

	if len(parts) == 0 {
		return ""
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
