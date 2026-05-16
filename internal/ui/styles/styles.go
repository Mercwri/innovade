package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Base background colors — exported so renderers can fill full-width rows.
	BgBase  = lipgloss.Color("#111827") // main content
	BgPanel = lipgloss.Color("#1F2937") // header, overlays, left pane

	// Foreground / accent palette
	colorAccent   = lipgloss.Color("#7C3AED")
	colorMuted    = lipgloss.Color("#6B7280")
	colorSubtle   = lipgloss.Color("#374151")
	colorSelected = lipgloss.Color("#1E1B4B")
	colorBorder = lipgloss.Color("#4B5563")
	ColorBorder = colorBorder // exported for use outside the styles package
	colorHeader   = lipgloss.Color("#F9FAFB")

	// Card colors
	colorBlue   = lipgloss.Color("#3B82F6")
	colorRed    = lipgloss.Color("#EF4444")
	colorGreen  = lipgloss.Color("#22C55E")
	colorWhite  = lipgloss.Color("#E5E7EB")
	colorPurple = lipgloss.Color("#A855F7")

	// ── Layout ───────────────────────────────────────────────────────────────

	StyleBase = lipgloss.NewStyle().
			Background(BgBase).
			Foreground(lipgloss.Color("#F9FAFB"))

	StyleHeader = lipgloss.NewStyle().
			Background(BgPanel).
			Foreground(colorHeader).
			Padding(0, 1).
			Bold(true)

	StyleHeaderMuted = StyleHeader.
				Foreground(colorMuted).
				Bold(false)

	StyleBorder = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorBorder)

	// ── List ─────────────────────────────────────────────────────────────────

	StyleColumnHeader = lipgloss.NewStyle().
				Background(BgBase).
				Foreground(colorMuted).
				Bold(true)

	StyleRowNormal = lipgloss.NewStyle().
			Background(BgBase).
			Foreground(lipgloss.Color("#D1D5DB"))

	StyleRowSelected = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	StyleRowDim = lipgloss.NewStyle().
			Background(BgBase).
			Foreground(colorSubtle)

	// ── Detail panel ─────────────────────────────────────────────────────────

	StyleDetailPanel = lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(colorBorder).
				Background(BgBase).
				Padding(0, 1)

	StyleDetailTitle = lipgloss.NewStyle().
				Background(BgBase).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	StyleDetailLabel = lipgloss.NewStyle().
				Background(BgBase).
				Foreground(colorMuted)

	StyleDetailValue = lipgloss.NewStyle().
				Background(BgBase).
				Foreground(lipgloss.Color("#D1D5DB"))

	StyleDetailDivider = lipgloss.NewStyle().
				Background(BgBase).
				Foreground(colorBorder)

	StyleDetailEffect = lipgloss.NewStyle().
				Background(BgBase).
				Foreground(lipgloss.Color("#E5E7EB")).
				Italic(true)

	StyleDetailBurst = lipgloss.NewStyle().
				Background(BgBase).
				Foreground(lipgloss.Color("#FCD34D"))

	// ── Command palette / overlays ────────────────────────────────────────────
	// Overlay items use BgPanel so they blend with the overlay background.

	StylePaletteOverlay = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Background(BgPanel).
				Padding(0, 1)

	StylePaletteInput = lipgloss.NewStyle().
				Background(BgPanel).
				Foreground(lipgloss.Color("#FFFFFF"))

	StylePaletteItem = lipgloss.NewStyle().
				Background(BgPanel).
				Foreground(lipgloss.Color("#D1D5DB")).
				Padding(0, 1)

	StylePaletteItemSelected = lipgloss.NewStyle().
					Background(colorAccent).
					Foreground(lipgloss.Color("#FFFFFF")).
					Padding(0, 1)

	// StylePaletteHint is for dim hint lines inside overlays (BgPanel context).
	StylePaletteHint = lipgloss.NewStyle().
				Background(BgPanel).
				Foreground(colorMuted).
				Italic(true)

	// StyleContentHint is for dim hint lines in the main content area (BgBase context).
	StyleContentHint = lipgloss.NewStyle().
				Background(BgBase).
				Foreground(colorMuted).
				Italic(true)

	// ── Rarity badges ─────────────────────────────────────────────────────────

	StyleRarityC  = lipgloss.NewStyle().Background(BgBase).Foreground(colorMuted)
	StyleRarityU  = lipgloss.NewStyle().Background(BgBase).Foreground(lipgloss.Color("#60A5FA"))
	StyleRarityR  = lipgloss.NewStyle().Background(BgBase).Foreground(lipgloss.Color("#A78BFA"))
	StyleRarityLR = lipgloss.NewStyle().Background(BgBase).Foreground(lipgloss.Color("#FCD34D")).Bold(true)
)

// RarityStyle returns the appropriate style for a rarity string.
func RarityStyle(rarity string) lipgloss.Style {
	switch rarity {
	case "U":
		return StyleRarityU
	case "R":
		return StyleRarityR
	case "LR":
		return StyleRarityLR
	default:
		return StyleRarityC
	}
}

// CardColorSwatch returns a colored dot for a card color name.
func CardColorSwatch(color string) string {
	dot := "●"
	switch color {
	case "Blue":
		return lipgloss.NewStyle().Background(BgBase).Foreground(colorBlue).Render(dot)
	case "Red":
		return lipgloss.NewStyle().Background(BgBase).Foreground(colorRed).Render(dot)
	case "Green":
		return lipgloss.NewStyle().Background(BgBase).Foreground(colorGreen).Render(dot)
	case "White":
		return lipgloss.NewStyle().Background(BgBase).Foreground(colorWhite).Render(dot)
	case "Purple":
		return lipgloss.NewStyle().Background(BgBase).Foreground(colorPurple).Render(dot)
	default:
		return lipgloss.NewStyle().Background(BgBase).Foreground(colorMuted).Render(dot)
	}
}
