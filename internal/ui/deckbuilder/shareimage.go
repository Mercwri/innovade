package deckbuilder

import (
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/webp"

	"github.com/Mercwri/innovade/internal/models"
	"github.com/Mercwri/innovade/internal/store"
	"github.com/Mercwri/innovade/internal/ui/library"
)

// Colors mirror internal/ui/styles/styles.go so the shared image matches the
// TUI's palette.
var (
	imgBgBase  = color.RGBA{0x11, 0x18, 0x27, 0xff}
	imgBgPanel = color.RGBA{0x1F, 0x29, 0x37, 0xff}
	imgBorder  = color.RGBA{0x4B, 0x55, 0x63, 0xff}
	imgTitle   = color.RGBA{0xFF, 0xFF, 0xFF, 0xff}
	imgHeader  = color.RGBA{0xF9, 0xFA, 0xFB, 0xff}
	imgMuted   = color.RGBA{0x9C, 0xA3, 0xAF, 0xff}
	imgValue   = color.RGBA{0xD1, 0xD5, 0xDB, 0xff}
)

var (
	regularFont, boldFont, monoFont *opentype.Font
)

func init() {
	var err error
	// goregular/gobold/gomono are embedded TTF byte slices from x/image; a
	// parse failure here would mean the library shipped corrupt data, so
	// there is nothing a caller could do about it — fail fast at startup.
	if regularFont, err = opentype.Parse(goregular.TTF); err != nil {
		panic(err)
	}
	if boldFont, err = opentype.Parse(gobold.TTF); err != nil {
		panic(err)
	}
	if monoFont, err = opentype.Parse(gomono.TTF); err != nil {
		panic(err)
	}
}

func newFace(f *opentype.Font, size float64) font.Face {
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(err)
	}
	return face
}

// imgCanvas is a thin wrapper around an RGBA buffer with text/shape helpers.
type imgCanvas struct {
	img *image.RGBA
}

func newImgCanvas(w, h int) *imgCanvas {
	c := &imgCanvas{img: image.NewRGBA(image.Rect(0, 0, w, h))}
	c.fill(c.img.Bounds(), imgBgBase)
	return c
}

func (c *imgCanvas) fill(r image.Rectangle, col color.Color) {
	stddraw.Draw(c.img, r, image.NewUniform(col), image.Point{}, stddraw.Src)
}

func (c *imgCanvas) strokeRect(r image.Rectangle, col color.Color, thickness int) {
	c.fill(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+thickness), col)
	c.fill(image.Rect(r.Min.X, r.Max.Y-thickness, r.Max.X, r.Max.Y), col)
	c.fill(image.Rect(r.Min.X, r.Min.Y, r.Min.X+thickness, r.Max.Y), col)
	c.fill(image.Rect(r.Max.X-thickness, r.Min.Y, r.Max.X, r.Max.Y), col)
}

// drawText draws s with its baseline at (x, y).
func (c *imgCanvas) drawText(x, y int, s string, face font.Face, col color.Color) {
	d := &font.Drawer{
		Dst:  c.img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}

func (c *imgCanvas) drawTextCentered(cx, y int, s string, face font.Face, col color.Color) {
	w := font.MeasureString(face, s).Round()
	c.drawText(cx-w/2, y, s, face, col)
}

// maxLineWidth returns the widest rendered width, in pixels, among all lines
// in all given slices — used to size a panel to fit its content exactly.
func maxLineWidth(face font.Face, lineSets ...[]string) int {
	max := 0
	for _, lines := range lineSets {
		for _, l := range lines {
			if w := font.MeasureString(face, l).Round(); w > max {
				max = w
			}
		}
	}
	return max
}

// drawImage copies src into dst at the given top-left, scaling to fit.
func (c *imgCanvas) drawImage(src image.Image, r image.Rectangle) {
	draw.CatmullRom.Scale(c.img, r, src, src.Bounds(), draw.Over, nil)
}

// ── Deck image export ────────────────────────────────────────────────────────

// exportImageCmd loads full card data for the deck, renders a shareable PNG
// (card grid + curve/burst/link panels), and writes it next to the app binary.
func exportImageCmd(deck models.Deck, s *store.Store, imageDir string) tea.Cmd {
	return func() tea.Msg {
		codes := make([]string, 0, len(deck.Entries))
		for _, e := range deck.Entries {
			codes = append(codes, e.CardCode)
		}
		cards, err := s.GetCardsByCodeMap(codes)
		if err != nil {
			return exportResultMsg{err: fmt.Errorf("load cards: %w", err)}
		}

		img := renderDeckImage(&deck, cards, imageDir)

		path := sanitizeFilename(deck.Name) + ".png"
		f, err := os.Create(path)
		if err != nil {
			return exportResultMsg{err: err}
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			return exportResultMsg{err: err}
		}
		return exportResultMsg{path: path}
	}
}

const (
	thumbW      = 132
	cellGap     = 16
	qtyLabelH   = 30
	imgMargin   = 28
	imgTitleH   = 60
	panelGap    = 20
	panelH      = 300
	panelMinW   = 240
	shieldCount = 6
)

var thumbH = int(math.Round(float64(thumbW) * 88 / 63))

// renderDeckImage lays out the deck as: title, a grid of card thumbnails with
// quantity counts, then a row of three panels (curve grid, burst chart, link
// table) — mirroring the analysis screen's data on one shareable image.
func renderDeckImage(deck *models.Deck, cards map[string]models.Card, imageDir string) image.Image {
	ensureArtDownloaded(deck, cards, imageDir)

	n := len(deck.Entries)
	if n == 0 {
		n = 1
	}
	cols := int(math.Ceil(math.Sqrt(float64(n) * 1.4)))
	if cols < 1 {
		cols = 1
	}
	if cols > n {
		cols = n
	}
	rows := int(math.Ceil(float64(n) / float64(cols)))

	cellW := thumbW + cellGap
	cellH := thumbH + qtyLabelH + cellGap
	gridW := cols*cellW + cellGap
	gridH := rows*cellH + cellGap

	curveLines := curveGridLines(deck, cards)
	burstPanelLines := burstLines(deck, cards)
	linkRows := computeLinkRows(deck, cards)
	linkLines := linkTableLines(linkRows, 10)

	bodyFace := newFace(monoFont, 13)
	const panelPad = 14
	minPanelW := maxLineWidth(bodyFace, curveLines, burstPanelLines, linkLines) + panelPad*2
	if minPanelW < panelMinW {
		minPanelW = panelMinW
	}

	canvasW := gridW + imgMargin*2
	if minW := minPanelW*3 + panelGap*2 + imgMargin*2; canvasW < minW {
		canvasW = minW
	}
	canvasH := imgMargin + imgTitleH + gridH + panelGap + panelH + imgMargin

	c := newImgCanvas(canvasW, canvasH)

	// ── Title ──────────────────────────────────────────────────────────────
	titleFace := newFace(boldFont, 32)
	baseline := imgMargin + titleFace.Metrics().Ascent.Ceil()
	c.drawText(imgMargin, baseline, deck.Name, titleFace, imgTitle)

	subFace := newFace(regularFont, 18)
	sub := fmt.Sprintf("%d / %d cards", deck.TotalCards(), models.DeckMaxSize)
	subW := font.MeasureString(subFace, sub).Round()
	c.drawText(canvasW-imgMargin-subW, baseline, sub, subFace, imgMuted)

	// ── Card grid ──────────────────────────────────────────────────────────
	gridTop := imgMargin + imgTitleH
	qtyFace := newFace(monoFont, 15)
	entries := sortedEntriesForExport(deck.Entries, cards)
	for i, e := range entries {
		col := i % cols
		row := i / cols
		x := imgMargin + cellGap + col*cellW
		y := gridTop + cellGap + row*cellH

		card := cards[e.CardCode]
		thumbRect := image.Rect(x, y, x+thumbW, y+thumbH)
		if thumb := loadCardThumb(imageDir, card); thumb != nil {
			c.drawImage(thumb, thumbRect)
		} else {
			c.drawPlaceholderThumb(thumbRect, card.CardCode)
		}
		c.strokeRect(thumbRect, imgBorder, 2)

		qty := fmt.Sprintf("%s ×%d", card.CardCode, e.Quantity)
		if card.HasBurst() {
			qty = "· " + qty
		}
		c.drawTextCentered(x+thumbW/2, y+thumbH+qtyLabelH-10, qty, qtyFace, imgValue)
	}

	// ── Bottom panels: curve grid · burst chart · link table ────────────────
	panelY := gridTop + gridH + panelGap
	panelW := (canvasW - imgMargin*2 - panelGap*2) / 3
	p1x := imgMargin
	p2x := p1x + panelW + panelGap
	p3x := p2x + panelW + panelGap

	c.drawPanel(p1x, panelY, panelW, panelH, "Curve", curveLines)
	c.drawPanel(p2x, panelY, panelW, panelH, "Burst", burstPanelLines)
	c.drawPanel(p3x, panelY, panelW, panelH, "Links", linkLines)

	return c.img
}

// exportCategoryOrder ranks entries for the shared image: Units, then Pilots,
// then Commands, then Bases (and anything else last), by card code within
// each group.
var exportCategoryOrder = map[models.Category]int{
	models.CategoryUnit:    0,
	models.CategoryPilot:   1,
	models.CategoryCommand: 2,
	models.CategoryBase:    3,
}

// sortedEntriesForExport returns deck entries ordered for the image grid,
// leaving deck.Entries (the player's arrangement) untouched.
func sortedEntriesForExport(entries []models.DeckEntry, cards map[string]models.Card) []models.DeckEntry {
	sorted := make([]models.DeckEntry, len(entries))
	copy(sorted, entries)

	rank := func(e models.DeckEntry) int {
		if r, ok := exportCategoryOrder[cards[e.CardCode].Category]; ok {
			return r
		}
		return len(exportCategoryOrder)
	}

	sort.Slice(sorted, func(i, j int) bool {
		ri, rj := rank(sorted[i]), rank(sorted[j])
		if ri != rj {
			return ri < rj
		}
		return sorted[i].CardCode < sorted[j].CardCode
	})
	return sorted
}

func (c *imgCanvas) drawPlaceholderThumb(r image.Rectangle, label string) {
	c.fill(r, imgBgPanel)
	face := newFace(regularFont, 13)
	c.drawTextCentered(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2, truncate(label, 12), face, imgMuted)
}

func (c *imgCanvas) drawPanel(x, y, w, h int, title string, lines []string) {
	r := image.Rect(x, y, x+w, y+h)
	c.fill(r, imgBgPanel)
	c.strokeRect(r, imgBorder, 2)

	const pad = 14
	titleFace := newFace(boldFont, 18)
	bodyFace := newFace(monoFont, 13)

	ty := y + pad + titleFace.Metrics().Ascent.Ceil()
	c.drawText(x+pad, ty, title, titleFace, imgHeader)

	divY := ty + titleFace.Metrics().Descent.Ceil() + 10
	c.fill(image.Rect(x+pad, divY, x+w-pad, divY+1), imgBorder)

	lh := bodyFace.Metrics().Height.Ceil()
	cy := divY + 10 + bodyFace.Metrics().Ascent.Ceil()
	for _, line := range lines {
		if cy > y+h-pad {
			break
		}
		c.drawText(x+pad, cy, line, bodyFace, imgValue)
		cy += lh
	}
}

// ensureArtDownloaded best-effort fetches any missing card art so the grid
// doesn't render placeholders for cards the library screen hasn't opened yet.
func ensureArtDownloaded(deck *models.Deck, cards map[string]models.Card, imageDir string) {
	if imageDir == "" {
		return
	}
	for _, e := range deck.Entries {
		card, ok := cards[e.CardCode]
		if !ok || card.DefaultImagePath == "" {
			continue
		}
		localPath := filepath.Join(imageDir, card.DefaultImagePath)
		if _, err := os.Stat(localPath); err == nil {
			continue
		}
		_ = library.DownloadArt(localPath, card.DefaultImagePath) // best-effort; placeholder shows on failure
	}
}

// loadCardThumb decodes the card's cached art file, if present.
func loadCardThumb(imageDir string, card models.Card) image.Image {
	if imageDir == "" || card.DefaultImagePath == "" {
		return nil
	}
	f, err := os.Open(filepath.Join(imageDir, card.DefaultImagePath))
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return img
}

// ── Curve grid panel ─────────────────────────────────────────────────────────

// curveGridLines produces the same level×category grid as the deckbuilder's
// left-pane curve display (renderCurveGrid), as plain text lines.
func curveGridLines(deck *models.Deck, cards map[string]models.Card) []string {
	const (
		minLv  = 1
		maxLv  = 9
		catW   = 4
		colW   = 3
		chartH = 4
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

	counts := [4][maxLv + 1]int{}
	var lvTotals [maxLv + 1]int
	if deck != nil {
		for _, e := range deck.Entries {
			card, ok := cards[e.CardCode]
			if !ok {
				continue
			}
			lv := card.Level
			if lv < 0 {
				lv = 0
			}
			if lv > maxLv {
				lv = maxLv
			}
			lvTotals[lv] += e.Quantity
			for ci, cat := range cats {
				if card.Category == cat.kind {
					counts[ci][lv] += e.Quantity
				}
			}
		}
	}

	var lines []string

	hdr := fmt.Sprintf("%-*s", catW, "")
	for lv := minLv; lv <= maxLv; lv++ {
		hdr += fmt.Sprintf("%*d", colW, lv)
	}
	hdr += fmt.Sprintf("%*s", colW, "Σ")
	lines = append(lines, hdr)

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
		lines = append(lines, row)
	}

	divLen := catW + (maxLv-minLv+1)*colW + colW
	lines = append(lines, strings.Repeat("-", divLen))

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
	lines = append(lines, totRow)

	maxV := 0
	for lv := minLv; lv <= maxLv; lv++ {
		if lvTotals[lv] > maxV {
			maxV = lvTotals[lv]
		}
	}
	for row := chartH; row >= 1; row-- {
		line := fmt.Sprintf("%-*s", catW, "")
		for lv := minLv; lv <= maxLv; lv++ {
			cell := " "
			if maxV > 0 && lvTotals[lv]*chartH/maxV >= row {
				cell = "#"
			}
			line += fmt.Sprintf("%*s", colW, cell)
		}
		lines = append(lines, line)
	}
	axis := fmt.Sprintf("%-*s", catW, "")
	for lv := minLv; lv <= maxLv; lv++ {
		axis += fmt.Sprintf("%*d", colW, lv)
	}
	lines = append(lines, axis)

	return lines
}

// ── Burst panel ──────────────────────────────────────────────────────────────

// burstLines mirrors the analysis screen's cumulative hypergeometric burst
// ladder: P(≥1) through P(≥shieldCount) burst cards among the shields.
func burstLines(deck *models.Deck, cards map[string]models.Card) []string {
	K := countBurstCards(deck, cards)
	N := models.DeckMaxSize

	lines := []string{fmt.Sprintf("Cards with burst:  %d / %d", K, N)}
	for m := 1; m <= shieldCount; m++ {
		p := shieldProbAtLeast(N, K, shieldCount, m)
		lines = append(lines, fmt.Sprintf("P(>=%d in shields): %5.1f%%", m, p*100))
	}
	return lines
}

func countBurstCards(deck *models.Deck, cards map[string]models.Card) int {
	if deck == nil {
		return 0
	}
	count := 0
	for _, e := range deck.Entries {
		if card, ok := cards[e.CardCode]; ok && card.HasBurst() {
			count += e.Quantity
		}
	}
	return count
}

// shieldProbAtLeast returns P(X>=m) where X is the number of burst cards drawn
// among s shields, from a deck of N cards containing K burst cards.
func shieldProbAtLeast(N, K, s, m int) float64 {
	if K <= 0 || N <= 0 {
		return 0
	}
	if K >= N {
		return 1
	}
	sum := 0.0
	for k := 0; k < m; k++ {
		sum += hyperGeomPMF(N, K, s, k)
	}
	p := 1 - sum
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

func hyperGeomPMF(N, K, s, k int) float64 {
	if k < 0 || k > s || k > K || s-k > N-K {
		return 0
	}
	return comb(K, k) * comb(N-K, s-k) / comb(N, s)
}

func comb(n, k int) float64 {
	if k < 0 || k > n {
		return 0
	}
	if k > n-k {
		k = n - k
	}
	result := 1.0
	for i := 0; i < k; i++ {
		result *= float64(n-i) / float64(i+1)
	}
	return result
}

// ── Link table panel ─────────────────────────────────────────────────────────

type linkRow struct {
	unitName, pilotName string
	unitQty, pilotQty   int
}

// computeLinkRows mirrors the analysis screen's computeLinks: unit/pilot pairs
// where the pilot satisfies the unit's link requirement by name or trait.
func computeLinkRows(deck *models.Deck, cards map[string]models.Card) []linkRow {
	if deck == nil || len(cards) == 0 {
		return nil
	}

	var units, pilots []models.Card
	for _, e := range deck.Entries {
		if e.Quantity == 0 {
			continue
		}
		card, ok := cards[e.CardCode]
		if !ok {
			continue
		}
		if card.Category == models.CategoryUnit && card.HasLinkRequirement() {
			units = append(units, card)
		}
		if card.IsPilotEligible() {
			pilots = append(pilots, card)
		}
	}

	var rows []linkRow
	seen := make(map[string]bool)

	for _, unit := range units {
		names, traits := unit.ParseLinkTerms()
		unitQty := deck.CardCount(unit.CardCode)

		for _, pilot := range pilots {
			key := unit.CardCode + "|" + pilot.CardCode
			if seen[key] {
				continue
			}

			linkName := pilot.Name
			if pilot.Category == models.CategoryCommand {
				if pn := pilot.ExtractPilotName(); pn != "" {
					linkName = pn
				}
			}

			matched := false
			for _, name := range names {
				if strings.EqualFold(name, linkName) {
					matched = true
					break
				}
			}
			if !matched {
				for _, trait := range traits {
					for _, t := range pilot.Types {
						if strings.EqualFold(trait, t) {
							matched = true
							break
						}
					}
					if matched {
						break
					}
				}
			}

			if matched {
				seen[key] = true
				rows = append(rows, linkRow{
					unitName:  unit.Name,
					pilotName: pilot.Name,
					unitQty:   unitQty,
					pilotQty:  deck.CardCount(pilot.CardCode),
				})
			}
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].unitName != rows[j].unitName {
			return rows[i].unitName < rows[j].unitName
		}
		return rows[i].pilotName < rows[j].pilotName
	})

	return rows
}

func linkTableLines(rows []linkRow, maxRows int) []string {
	if len(rows) == 0 {
		return []string{"No linked pairs found in deck."}
	}

	const nameW, qW = 13, 3
	lines := []string{
		fmt.Sprintf("%-*s %-*s %*s %*s", nameW, "Unit", nameW, "Pilot", qW, "xU", qW, "xP"),
	}

	shown := len(rows)
	if maxRows > 0 && shown > maxRows {
		shown = maxRows
	}
	for _, r := range rows[:shown] {
		lines = append(lines, fmt.Sprintf("%-*s %-*s %*d %*d",
			nameW, truncate(r.unitName, nameW),
			nameW, truncate(r.pilotName, nameW),
			qW, r.unitQty,
			qW, r.pilotQty,
		))
	}
	if shown < len(rows) {
		lines = append(lines, fmt.Sprintf("+%d more", len(rows)-shown))
	}
	return lines
}
