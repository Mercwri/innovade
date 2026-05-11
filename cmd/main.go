package main

import (
	"fmt"
	"os"

	"github.com/Mercwri/innovade/internal/store"
	"github.com/Mercwri/innovade/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	s, err := store.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize store: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	app, err := ui.New(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize app: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "application error: %v\n", err)
		os.Exit(1)
	}
}
