package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cardinal9985/flashgrab/internal/download"
	"github.com/cardinal9985/flashgrab/internal/sites"
)

type doneModel struct {
	game    *sites.Game
	results []*download.Result
	errors  []error
	width   int
}

func newDoneModel(game *sites.Game, results []*download.Result, errors []error) doneModel {
	return doneModel{
		game:    game,
		results: results,
		errors:  errors,
	}
}

type restartMsg struct{}

func (m doneModel) Init() tea.Cmd { return nil }

func (m doneModel) Update(msg tea.Msg) (doneModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter", "n":
			return m, func() tea.Msg { return restartMsg{} }
		}
	}

	return m, nil
}

func (m doneModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(m.game.Title) + "\n")
	b.WriteString(dimStyle.Render("from "+m.game.Source) + "\n")

	if fp := m.game.Flashpoint; fp != nil {
		info := fmt.Sprintf("Flashpoint: %q", fp.Title)
		if fp.Developer != "" {
			info += " by " + fp.Developer
		}
		if fp.Platform != "" {
			info += " (" + fp.Platform + ")"
		}
		b.WriteString(dimStyle.Render(info) + "\n")
	}

	b.WriteString("\n")

	for _, r := range m.results {
		icon := successStyle.Render("[saved]")
		if r.Existed {
			icon = warnStyle.Render("[skip] ")
		}
		b.WriteString(fmt.Sprintf("  %s %s  %s\n",
			icon,
			r.Filename,
			dimStyle.Render(formatSize(r.Size)),
		))
		b.WriteString(dimStyle.Render(fmt.Sprintf("         %s", r.Path)) + "\n")
	}

	for _, e := range m.errors {
		b.WriteString(fmt.Sprintf("  %s %s\n", errorStyle.Render("[err] "), e.Error()))
	}

	saved := 0
	skipped := 0
	for _, r := range m.results {
		if r.Existed {
			skipped++
		} else {
			saved++
		}
	}

	b.WriteString("\n")
	summary := boxStyle.Render(fmt.Sprintf(
		"%s saved, %s skipped, %s failed",
		successStyle.Render(fmt.Sprintf("%d", saved)),
		warnStyle.Render(fmt.Sprintf("%d", skipped)),
		errorStyle.Render(fmt.Sprintf("%d", len(m.errors))),
	))
	b.WriteString(summary + "\n")

	b.WriteString("\n" + helpStyle.Render("n/enter: download another  q: quit"))

	return b.String()
}
