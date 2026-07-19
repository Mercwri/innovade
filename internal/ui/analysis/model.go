package analysis

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Mercwri/innovade/internal/models"
	"github.com/Mercwri/innovade/internal/store"
	"github.com/Mercwri/innovade/internal/ui/styles"
)

type linkPair struct {
	unit     models.Card
	pilot    models.Card
	unitQty  int
	pilotQty int
}

type cardsLoadedMsg struct {
	cards map[string]models.Card
	err   error
}

// Model is the analysis Bubble Tea model.
type Model struct {
	store *store.Store
	deck  *models.Deck
	cards map[string]models.Card

	links   []linkPair
	bursts  int
	hand    []models.Card
	shields []models.Card

	width  int
	height int
	loaded bool
	err    error
}

func New(s *store.Store) Model {
	return Model{store: s}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetDeck updates the active deck and triggers an async card load.
func (m *Model) SetDeck(deck *models.Deck) tea.Cmd {
	m.deck = deck
	m.loaded = false
	m.err = nil
	m.hand = nil
	m.shields = nil
	m.links = nil
	m.bursts = 0

	if deck == nil {
		m.cards = nil
		m.loaded = true
		return nil
	}

	codes := make([]string, 0, len(deck.Entries))
	for _, e := range deck.Entries {
		codes = append(codes, e.CardCode)
	}
	s := m.store
	return func() tea.Msg {
		cards, err := s.GetCardsByCodeMap(codes)
		return cardsLoadedMsg{cards: cards, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case cardsLoadedMsg:
		m.loaded = true
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.cards = msg.cards
		m.links = computeLinks(m.deck, m.cards)
		m.bursts = countBursts(m.deck, m.cards)
		return m, nil

	case tea.KeyMsg:
		if m.deck != nil && m.loaded && m.err == nil {
			switch msg.String() {
			case "d":
				m.hand = drawHand(m.deck, m.cards, 5)
			case "s":
				m.shields = drawHand(m.deck, m.cards, 6)
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.deck == nil {
		return styles.StyleDetailLabel.Render("No deck selected. Activate a deck from the Deck Builder.")
	}
	if !m.loaded {
		return "Loading analysis..."
	}
	if m.err != nil {
		return fmt.Sprintf("error: %v", m.err)
	}

	leftW := m.width * 55 / 100
	rightW := m.width - leftW

	leftBox := lipgloss.NewStyle().Width(leftW).Height(m.height).Background(styles.BgBase).
		Render(m.renderLeft(leftW))
	rightBox := lipgloss.NewStyle().Width(rightW).Height(m.height).Background(styles.BgBase).
		Render(m.renderRight(rightW))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
}

func (m Model) renderLeft(w int) string {
	var sb strings.Builder
	div := styles.StyleDetailDivider.Render(strings.Repeat("─", w))

	sb.WriteString(styles.StyleDetailTitle.Render("Curve"))
	sb.WriteString("\n")
	sb.WriteString(div)
	sb.WriteString("\n")
	sb.WriteString(m.renderGrid())
	sb.WriteString(div)
	sb.WriteString("\n")
	sb.WriteString(m.renderBursts())

	return sb.String()
}

func (m Model) renderRight(w int) string {
	var sb strings.Builder
	div := styles.StyleDetailDivider.Render(strings.Repeat("─", w))

	sb.WriteString(styles.StyleDetailTitle.Render("Links"))
	sb.WriteString("\n")
	sb.WriteString(div)
	sb.WriteString("\n")
	sb.WriteString(m.renderLinks(w))
	sb.WriteString("\n")
	sb.WriteString(div)
	sb.WriteString("\n")
	sb.WriteString(styles.StyleDetailTitle.Render("Sample Hand"))
	sb.WriteString("\n")
	sb.WriteString(m.renderHand(w))
	sb.WriteString("\n")
	sb.WriteString(div)
	sb.WriteString("\n")
	sb.WriteString(styles.StyleDetailTitle.Render("Shields  "))
	sb.WriteString(styles.StyleDetailBurst.Render("● burst"))
	sb.WriteString("\n")
	sb.WriteString(m.renderShields(w))

	return sb.String()
}

// renderGrid draws the level × category grid identical to the deckbuilder's curve display.
func (m Model) renderGrid() string {
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

	if m.deck != nil {
		for _, e := range m.deck.Entries {
			card, ok := m.cards[e.CardCode]
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

	var sb strings.Builder

	hdr := fmt.Sprintf("%-*s", catW, "")
	for lv := minLv; lv <= maxLv; lv++ {
		hdr += fmt.Sprintf("%*d", colW, lv)
	}
	hdr += fmt.Sprintf("%*s", colW, "Σ")
	sb.WriteString(styles.StyleColumnHeader.Render(hdr))
	sb.WriteString("\n")

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
	axis := fmt.Sprintf("%-*s", catW, "")
	for lv := minLv; lv <= maxLv; lv++ {
		axis += fmt.Sprintf("%*d", colW, lv)
	}
	sb.WriteString(styles.StyleDetailLabel.Render(axis))
	sb.WriteString("\n")

	return sb.String()
}

// renderBursts shows burst card count and the cumulative hypergeometric
// probability of drawing at least m burst cards among all 6 shields, for
// every threshold m from 1 to 6.
func (m Model) renderBursts() string {
	const shieldCount = 6

	K := m.bursts
	N := models.DeckMaxSize

	var sb strings.Builder
	sb.WriteString(styles.StyleDetailTitle.Render("Burst"))
	sb.WriteString("\n")
	sb.WriteString(styles.StyleDetailValue.Render(
		fmt.Sprintf("Cards with burst:  %d / %d", K, N),
	))
	sb.WriteString("\n")

	for m := 1; m <= shieldCount; m++ {
		prob := shieldProbAtLeast(N, K, shieldCount, m)
		sb.WriteString(styles.StyleDetailValue.Render(
			fmt.Sprintf("P(≥%d in shields):  %5.1f%%", m, prob*100),
		))
		sb.WriteString("\n")
	}
	return sb.String()
}

// renderLinks renders the unit/pilot link pair table.
func (m Model) renderLinks(w int) string {
	if len(m.links) == 0 {
		return styles.StyleDetailLabel.Render("No linked pairs found in deck.\n")
	}

	const qW = 3
	// Two name columns + 2 qty columns + 3 separating spaces
	nameW := (w - 2*qW - 3) / 2
	if nameW < 4 {
		nameW = 4
	}

	var sb strings.Builder
	header := fmt.Sprintf("%-*s %-*s %*s %*s", nameW, "Unit", nameW, "Pilot", qW, "×U", qW, "×P")
	sb.WriteString(styles.StyleColumnHeader.Render(header))
	sb.WriteString("\n")

	for _, lp := range m.links {
		row := fmt.Sprintf("%-*s %-*s %*d %*d",
			nameW, truncate(lp.unit.Name, nameW),
			nameW, truncate(lp.pilot.Name, nameW),
			qW, lp.unitQty,
			qW, lp.pilotQty,
		)
		sb.WriteString(styles.StyleDetailValue.Render(row))
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderHand renders the drawn hand or a draw prompt.
func (m Model) renderHand(w int) string {
	if len(m.hand) == 0 {
		return styles.StyleContentHint.Render("Press [d] to draw a sample 5-card hand\n")
	}

	var sb strings.Builder
	sb.WriteString(styles.StyleContentHint.Render("Press [d] to redraw"))
	sb.WriteString("\n\n")

	for _, card := range m.hand {
		sb.WriteString(styles.StyleDetailTitle.Render(truncate(card.Name, w-1)))
		sb.WriteString("\n")

		info := catAbbrev(card.Category)
		if card.Level > 0 {
			info += fmt.Sprintf(" · Lv%d", card.Level)
		}
		if card.Cost > 0 {
			info += fmt.Sprintf(" · Cost%d", card.Cost)
		}
		if card.AP > 0 || card.HP > 0 {
			info += fmt.Sprintf(" · %d/%d", card.AP, card.HP)
		}
		sb.WriteString(styles.StyleDetailLabel.Render(info))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// renderShields renders the 6-card shield row with burst cards highlighted.
func (m Model) renderShields(w int) string {
	if len(m.shields) == 0 {
		return styles.StyleContentHint.Render("Press [s] to draw sample shields\n")
	}

	var sb strings.Builder
	sb.WriteString(styles.StyleContentHint.Render("Press [s] to redraw"))
	sb.WriteString("\n\n")

	for _, card := range m.shields {
		if card.HasBurst() {
			sb.WriteString(styles.StyleDetailBurst.Bold(true).Render("● " + truncate(card.Name, w-3)))
		} else {
			sb.WriteString(styles.StyleDetailValue.Render("  " + truncate(card.Name, w-3)))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	return sb.String()
}

// computeLinks finds all unit/pilot pairs where the pilot satisfies the unit's
// link requirement by name or type trait.
func computeLinks(deck *models.Deck, cards map[string]models.Card) []linkPair {
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
		if (card.Category == models.CategoryUnit) && card.HasLinkRequirement() {
			units = append(units, card)
		}
		if card.IsPilotEligible() {
			pilots = append(pilots, card)
		}
	}

	var pairs []linkPair
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
				pairs = append(pairs, linkPair{
					unit:     unit,
					pilot:    pilot,
					unitQty:  unitQty,
					pilotQty: deck.CardCount(pilot.CardCode),
				})
			}
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].unit.Name != pairs[j].unit.Name {
			return pairs[i].unit.Name < pairs[j].unit.Name
		}
		return pairs[i].pilot.Name < pairs[j].pilot.Name
	})

	return pairs
}

// countBursts counts the total number of burst-capable cards in the deck,
// expanded by quantity (so 4×copies of a burst card counts as 4).
func countBursts(deck *models.Deck, cards map[string]models.Card) int {
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

// shieldProbAtLeast returns P(X≥m) where X is the number of burst cards drawn
// among s shields, from a deck of N cards containing K burst cards, using the
// hypergeometric distribution.
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

// hyperGeomPMF returns P(X=k) for X ~ Hypergeometric(N, K, s): the probability
// of drawing exactly k successes in s draws without replacement from a
// population of N containing K successes.
func hyperGeomPMF(N, K, s, k int) float64 {
	if k < 0 || k > s || k > K || s-k > N-K {
		return 0
	}
	return comb(K, k) * comb(N-K, s-k) / comb(N, s)
}

// comb returns the binomial coefficient C(n, k) as a float64.
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

// drawHand builds the full deck pool (each card repeated by quantity), shuffles
// it, and returns the first n cards as a sample hand.
func drawHand(deck *models.Deck, cards map[string]models.Card, n int) []models.Card {
	if deck == nil {
		return nil
	}
	var pool []models.Card
	for _, e := range deck.Entries {
		if card, ok := cards[e.CardCode]; ok {
			for i := 0; i < e.Quantity; i++ {
				pool = append(pool, card)
			}
		}
	}
	if len(pool) == 0 {
		return nil
	}
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	if n > len(pool) {
		n = len(pool)
	}
	return pool[:n]
}

func catAbbrev(cat models.Category) string {
	switch cat {
	case models.CategoryUnit:
		return "Unit"
	case models.CategoryCommand:
		return "Command"
	case models.CategoryPilot:
		return "Pilot"
	case models.CategoryBase:
		return "Base"
	case models.CategoryUnitToken:
		return "Token"
	}
	return string(cat)
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
