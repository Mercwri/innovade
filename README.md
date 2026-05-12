# INNOVADE

A terminal-based deck builder for the Gundam Card Game, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Card Library** — browse and filter cards by color, category, set, and more
- **Deck Builder** — create and manage decks with a live cost-curve chart
- **Import** — load a deck from a `.txt` file or a `deckbuilder.egmanevents.com` share URL
- **Export** — save to `.txt`, copy the decklist to clipboard, or copy a [Mobile Suit Arena](https://mobilesuitarena.com) share link

## Requirements

- Go 1.25+
- A `data/cards.json` card data file in the working directory

## Build & Run

```sh
go build -o innovade ./cmd
./innovade
```

On first run, open the command palette (`?`) and select **Import Cards** to load `data/cards.json` into the local database.

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `⇧1` | Card Library |
| `⇧2` | Deck Builder |
| `?` / `Ctrl+P` | Command palette |
| `q` / `Ctrl+C` | Quit |

### Deck Builder

| Key | Action |
|-----|--------|
| `↑` / `k`, `↓` / `j` | Navigate decks |
| `Enter` | Activate deck for editing |
| `n` | New deck |
| `r` | Rename selected deck |
| `x` | Delete selected deck |
| `e` | Export (file / clipboard / MSA link) |
| `i` | Import (`.txt` file or URL) |
| `PgUp` / `Ctrl+U` | Scroll card list up |
| `PgDn` / `Ctrl+D` | Scroll card list down |

### Card Library

| Key | Action |
|-----|--------|
| `↑` / `k`, `↓` / `j` | Navigate cards |
| `Enter` | Add card to active deck |
| `-` | Remove card from active deck |
| `f` | Filter palette |
| `Esc` | Close detail / clear filter |

## License

MIT — see [LICENSE](LICENSE).
