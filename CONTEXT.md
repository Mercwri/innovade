# Innovade — Project Context

## What this is
A TUI-based Gundam Card Game platform in Go.
Module: github.com/Mercwri/innovade

## Stack
- Bubble Tea (TUI framework)
- lipgloss + bubbles (styling + components)
- modernc.org/sqlite (no CGO SQLite)
- google/uuid

## What's been built
- internal/models/ — Card, Deck, CardFilter, CardQuery types
- internal/store/ — Store struct, SQLite migrations (embed.FS), 
  QueryCards, GetCard, ListDecks, SaveDeck, DeleteDeck, ImportCardsFromJSON
- internal/ui/styles.go — lipgloss style definitions
- internal/ui/keys.go — shared key bindings
- internal/ui/app.go — root AppModel, view switching, palette overlay
- internal/ui/palette.go — global command palette with fuzzy filtering
- internal/ui/library/model.go — card library model
- internal/ui/library/list.go — list rendering
- internal/ui/library/detail.go — card detail panel

## Current task
Building the Card Library UI. Next steps are:
1. library/palette.go — filter palette for the library
2. cmd/main.go — entry point wiring store + TUI together
3. Fix the renderWithFilterPalette function in list.go (has a dead code bug)

## Key decisions
- Navigation is command-palette driven (? or : to open)
- Card list columns: ID | Name | Lv/Cost | AP/HP | Color
- Detail panel sits below the list (top/bottom split, 2/3 list / 1/3 detail)
- Deck size: exactly 50 cards, max 4 copies per card, 2 for restricted
- DB lives in OS user data dir (~/.local/share/innovade/ on Linux)
- Migrations embedded in binary via go:embed