package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Color palette
	colorAccent   = lipgloss.Color("#7C3AED") // purple
	colorMuted    = lipgloss.Color("#6B7280")
	colorSubtle   = lipgloss.Color("#374151")
	colorSelected = lipgloss.Color("#1E1B4B")
	colorBorder   = lipgloss.Color("#4B5563")
	colorHeader   = lipgloss.Color("#F9FAFB")

	// Card colors
	colorBlue   = lipgloss.Color("#3B82F6")
	colorRed    = lipgloss.Color("#EF4444")
	colorGreen  = lipgloss.Color("#22C55E")
	colorWhite  = lipgloss.Color("#E5E7EB")
	colorPurple = lipgloss.Color("#A855F7")

	// Layout
	StyleBase = lipgloss.NewStyle().
			Background(lipgloss.Color("#111827")).
			Foreground(lipgloss.Color("#F9FAFB"))

	StyleHeader = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(colorHeader).
			Padding(0, 1).
			Bold(true)

	StyleHeaderMuted = StyleHeader.
				Foreground(colorMuted).
				Bold(false)

	StyleBorder = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorBorder)

	// List
	StyleColumnHeader = lipgloss.NewStyle().
				Foreground(colorMuted).
				Bold(true)

	StyleRowNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	StyleRowSelected = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	StyleRowDim = lipgloss.NewStyle().
			Foreground(colorSubtle)

	// Detail panel
	StyleDetailPanel = lipgloss.NewStyle().
				BorderTop(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(colorBorder).
				Padding(0, 1)

	StyleDetailTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	StyleDetailLabel = lipgloss.NewStyle().
				Foreground(colorMuted)

	StyleDetailValue = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D1D5DB"))

	StyleDetailDivider = lipgloss.NewStyle().
				Foreground(colorBorder)

	StyleDetailEffect = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E5E7EB")).
				Italic(true)

	StyleDetailBurst = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FCD34D"))

	// Command palette
	StylePaletteOverlay = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Background(lipgloss.Color("#1F2937")).
				Padding(0, 1)

	StylePaletteInput = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))

	StylePaletteItem = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D1D5DB")).
				Padding(0, 1)

	StylePaletteItemSelected = lipgloss.NewStyle().
					Background(colorAccent).
					Foreground(lipgloss.Color("#FFFFFF")).
					Padding(0, 1)

	StylePaletteHint = lipgloss.NewStyle().
				Foreground(colorMuted).
				Italic(true)

	// Rarity badges
	StyleRarityC  = lipgloss.NewStyle().Foreground(colorMuted)
	StyleRarityU  = lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA"))
	StyleRarityR  = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
	StyleRarityLR = lipgloss.NewStyle().Foreground(lipgloss.Color("#FCD34D")).Bold(true)
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
		return lipgloss.NewStyle().Foreground(colorBlue).Render(dot)
	case "Red":
		return lipgloss.NewStyle().Foreground(colorRed).Render(dot)
	case "Green":
		return lipgloss.NewStyle().Foreground(colorGreen).Render(dot)
	case "White":
		return lipgloss.NewStyle().Foreground(colorWhite).Render(dot)
	case "Purple":
		return lipgloss.NewStyle().Foreground(colorPurple).Render(dot)
	default:
		return lipgloss.NewStyle().Foreground(colorMuted).Render(dot)
	}
}
