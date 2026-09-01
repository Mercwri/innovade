// Package curve computes and renders the level × category "curve" grid used
// across the deck builder, analysis view, and card library deck panel.
package curve

import (
	"fmt"
	"strings"

	"github.com/Mercwri/innovade/internal/models"
	"github.com/Mercwri/innovade/internal/ui/styles"
)

const (
	// MinLevel and MaxLevel bound the columns of the grid. Anything above
	// MaxLevel is clamped into the last column.
	MinLevel = 1
	MaxLevel = 9

	catW = 4 // label column width
	colW = 3 // per-level and Σ column width

	// GridWidth is the rendered width of a grid/bar line, in cells.
	GridWidth = catW + (MaxLevel-MinLevel+1)*colW + colW
)

// StatFunc resolves a card code to the fields the curve needs. ok is false
// when the code is unknown, in which case the entry is skipped.
type StatFunc func(cardCode string) (category models.Category, level int, ok bool)

var (
	catKinds  = [4]models.Category{models.CategoryUnit, models.CategoryCommand, models.CategoryPilot, models.CategoryBase}
	catLabels = [4]string{"Unt", "Cmd", "Plt", "Bas"}
)

// Data holds the tallied counts for a deck.
type Data struct {
	counts   [4][MaxLevel + 1]int
	lvTotals [MaxLevel + 1]int
	grand    int
}

// Compute tallies the given deck entries by category and level.
func Compute(entries []models.DeckEntry, stat StatFunc) Data {
	var d Data
	for _, e := range entries {
		cat, lv, ok := stat(e.CardCode)
		if !ok {
			continue
		}
		if lv < 0 {
			lv = 0
		}
		if lv > MaxLevel {
			lv = MaxLevel
		}
		d.lvTotals[lv] += e.Quantity
		d.grand += e.Quantity
		for ci, kind := range catKinds {
			if cat == kind {
				d.counts[ci][lv] += e.Quantity
			}
		}
	}
	return d
}

// Total returns the number of cards counted across all categories/levels.
func (d Data) Total() int { return d.grand }

// Grid renders the header row, one row per category, a divider, and a totals
// row — 7 newline-terminated lines.
func (d Data) Grid() string {
	var sb strings.Builder

	hdr := fmt.Sprintf("%-*s", catW, "")
	for lv := MinLevel; lv <= MaxLevel; lv++ {
		hdr += fmt.Sprintf("%*d", colW, lv)
	}
	hdr += fmt.Sprintf("%*s", colW, "Σ")
	sb.WriteString(styles.StyleColumnHeader.Render(hdr))
	sb.WriteString("\n")

	for ci, label := range catLabels {
		total := 0
		row := fmt.Sprintf("%-*s", catW, label)
		for lv := MinLevel; lv <= MaxLevel; lv++ {
			n := d.counts[ci][lv]
			total += n
			row += cell(n)
		}
		row += cell(total)
		sb.WriteString(styles.StyleDetailLabel.Render(row))
		sb.WriteString("\n")
	}

	sb.WriteString(styles.StyleDetailDivider.Render(strings.Repeat("─", GridWidth)))
	sb.WriteString("\n")

	totRow := fmt.Sprintf("%-*s", catW, "Tot")
	for lv := MinLevel; lv <= MaxLevel; lv++ {
		totRow += cell(d.lvTotals[lv])
	}
	totRow += cell(d.grand)
	sb.WriteString(styles.StyleDetailValue.Render(totRow))
	sb.WriteString("\n")

	return sb.String()
}

// Bars renders a level-distribution bar chart `height` rows tall followed by
// an axis row — height+1 newline-terminated lines.
func (d Data) Bars(height int) string {
	maxV := 0
	for lv := MinLevel; lv <= MaxLevel; lv++ {
		if d.lvTotals[lv] > maxV {
			maxV = d.lvTotals[lv]
		}
	}

	var sb strings.Builder
	for row := height; row >= 1; row-- {
		line := fmt.Sprintf("%-*s", catW, "")
		for lv := MinLevel; lv <= MaxLevel; lv++ {
			cellStr := " "
			if maxV > 0 && d.lvTotals[lv]*height/maxV >= row {
				cellStr = "█"
			}
			line += fmt.Sprintf("%*s", colW, cellStr)
		}
		sb.WriteString(styles.StyleDetailValue.Render(line))
		sb.WriteString("\n")
	}

	axis := fmt.Sprintf("%-*s", catW, "")
	for lv := MinLevel; lv <= MaxLevel; lv++ {
		axis += fmt.Sprintf("%*d", colW, lv)
	}
	sb.WriteString(styles.StyleDetailLabel.Render(axis))
	sb.WriteString("\n")
	return sb.String()
}

func cell(n int) string {
	if n == 0 {
		return fmt.Sprintf("%*s", colW, "·")
	}
	return fmt.Sprintf("%*d", colW, n)
}
