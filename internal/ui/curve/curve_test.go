package curve

import (
	"strings"
	"testing"

	"github.com/Mercwri/innovade/internal/models"
)

func TestComputeAndGrid(t *testing.T) {
	entries := []models.DeckEntry{
		{CardCode: "u1", Quantity: 3}, // Unit lv2
		{CardCode: "u2", Quantity: 2}, // Unit lv4
		{CardCode: "c1", Quantity: 4}, // Command lv1
		{CardCode: "p1", Quantity: 1}, // Pilot lv0 -> clamped column skipped (lv 0 not shown)
		{CardCode: "x1", Quantity: 2}, // unknown -> skipped
		{CardCode: "b1", Quantity: 1}, // Base lv12 -> clamped to 9
	}
	stat := func(code string) (models.Category, int, bool) {
		switch code {
		case "u1":
			return models.CategoryUnit, 2, true
		case "u2":
			return models.CategoryUnit, 4, true
		case "c1":
			return models.CategoryCommand, 1, true
		case "p1":
			return models.CategoryPilot, 0, true
		case "b1":
			return models.CategoryBase, 12, true
		}
		return "", 0, false
	}

	d := Compute(entries, stat)

	if got, want := d.Total(), 3+2+4+1+1; got != want {
		t.Fatalf("Total() = %d, want %d", got, want)
	}
	if d.counts[0][2] != 3 || d.counts[0][4] != 2 {
		t.Errorf("unit counts wrong: %v", d.counts[0])
	}
	if d.counts[3][MaxLevel] != 1 {
		t.Errorf("base lv12 not clamped to %d: %v", MaxLevel, d.counts[3])
	}
	if d.lvTotals[0] != 1 {
		t.Errorf("lv0 total = %d, want 1", d.lvTotals[0])
	}

	grid := d.Grid()
	lines := strings.Split(strings.TrimRight(grid, "\n"), "\n")
	if len(lines) != 7 { // header + 4 cats + divider + Tot
		t.Fatalf("Grid() = %d lines, want 7:\n%s", len(lines), grid)
	}
	if !strings.HasPrefix(lines[1], "Unt") {
		t.Errorf("row 1 = %q, want Unt…", lines[1])
	}
	if !strings.HasPrefix(lines[6], "Tot") {
		t.Errorf("row 6 = %q, want Tot…", lines[6])
	}
}

func TestBarsLineCount(t *testing.T) {
	d := Compute(nil, func(string) (models.Category, int, bool) { return "", 0, false })
	bars := d.Bars(4)
	lines := strings.Split(strings.TrimRight(bars, "\n"), "\n")
	if len(lines) != 5 { // 4 bar rows + axis
		t.Fatalf("Bars(4) = %d lines, want 5", len(lines))
	}
}
