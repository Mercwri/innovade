package keys

import "github.com/charmbracelet/bubbles/key"

// GlobalKeys are active from any view.
type GlobalKeys struct {
	Palette key.Binding
	Quit    key.Binding
}

var Global = GlobalKeys{
	Palette: key.NewBinding(
		key.WithKeys("?", ":"),
		key.WithHelp("?", "command palette"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// LibraryKeys are active when the card library is focused.
type LibraryKeys struct {
	Up             key.Binding
	Down           key.Binding
	Top            key.Binding
	Bottom         key.Binding
	PageUp         key.Binding
	PageDown       key.Binding
	AddToDeck      key.Binding
	RemoveFromDeck key.Binding
	FindLinks      key.Binding
}

var Library = LibraryKeys{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Top: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "top"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "bottom"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup", "ctrl+u"),
		key.WithHelp("PgUp", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown", "ctrl+d"),
		key.WithHelp("PgDn", "page down"),
	),
	AddToDeck: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "add to deck"),
	),
	RemoveFromDeck: key.NewBinding(
		key.WithKeys("delete", "backspace"),
		key.WithHelp("del", "remove from deck"),
	),
	FindLinks: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "find links"),
	),
}

// PaletteKeys are active when the command palette is open.
type PaletteKeys struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Close  key.Binding
}

var Palette = PaletteKeys{
	Up: key.NewBinding(
		key.WithKeys("up", "ctrl+p"),
		key.WithHelp("↑", "previous"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "ctrl+n"),
		key.WithHelp("↓", "next"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Close: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close"),
	),
}
