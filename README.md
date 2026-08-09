# INNOVADE

A terminal-based deck builder for the Gundam Card Game, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Card Library** — browse and filter cards by color, category, set, and more; jump between a unit and its eligible pilots (or a pilot and its eligible units) with the link finder
- **Deck Builder** — create, rename, and clone decks with a live cost-curve chart
- **Analysis** — check deck legality against ban/limitation rules, review linked unit/pilot pairs, and draw a sample opening hand or shield row
- **Import** — load a deck from a `.txt` file or a `deckbuilder.egmanevents.com` share URL
- **Export** — save to `.txt`, copy the decklist to clipboard, copy a [Mobile Suit Arena](https://mobilesuitarena.com) share link, or save a PNG deck image

## Requirements

- Go 1.25+
- A `data/cards.json` card data file in the working directory (external export; not tracked/produced by this repo)
- Optionally, a `data/bans.json` file describing deck limitation/ban rules

## Build & Run

```sh
go build -o innovade ./cmd
./innovade
```

On first run, open the command palette (`?`) and select **Import Cards** to load `data/cards.json` into the local database (`~/.local/share/innovade/` on Linux). Re-running the import is safe — it upserts by card code.

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `⇧1` | Card Library |
| `⇧2` | Deck Builder |
| `?` / `Ctrl+P` | Command palette (also reaches Analysis) |
| `q` / `Ctrl+C` | Quit |

### Deck Builder

| Key | Action |
|-----|--------|
| `↑` / `k`, `↓` / `j` | Navigate decks |
| `Enter` | Activate deck for editing |
| `n` | New deck |
| `r` | Rename selected deck |
| `c` | Clone selected deck |
| `x` | Delete selected deck |
| `e` | Export (`.txt` / clipboard / MSA link / PNG image) |
| `i` | Import (`.txt` file or URL) |
| `PgUp` / `Ctrl+U` | Scroll card list up |
| `PgDn` / `Ctrl+D` | Scroll card list down |

### Card Library

| Key | Action |
|-----|--------|
| `↑` / `k`, `↓` / `j` | Navigate cards |
| `g` / `G` | Jump to top / bottom |
| `←` / `→` | Focus library / focus deck panel |
| `Enter` | Add card to active deck |
| `Del` / `Backspace` | Remove card from active deck |
| `t` | Text search |
| `f` | Filter palette |
| `l` | Find links (matching pilots for a unit, or units for a pilot) |
| `c` | Clear filters |
| `Esc` | Close detail / clear filter |

### Analysis

| Key | Action |
|-----|--------|
| `d` | Draw a sample 5-card opening hand |
| `s` | Draw a sample 6-card shield row |

## License

MIT — see [LICENSE](LICENSE).
