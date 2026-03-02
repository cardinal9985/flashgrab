package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cardinal9985/flashgrab/internal/sites"
)

// pickModel lets the user choose which file(s) to download when a game has
// multiple uploads (common on itch.io).
type pickModel struct {
	game     *sites.Game
	cursor   int
	selected map[int]bool
	width    int
}

func newPickModel(game *sites.Game) pickModel {
	sel := make(map[int]bool)
	// Pre-select everything if there's only one file.
	if len(game.Files) == 1 {
		sel[0] = true
	}
	return pickModel{
		game:     game,
		selected: sel,
	}
}

// pickDoneMsg carries the files the user chose to download.
type pickDoneMsg struct {
	game  *sites.Game
	files []sites.GameFile
}

func (m pickModel) Init() tea.Cmd { return nil }

func (m pickModel) Update(msg tea.Msg) (pickModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.game.Files)-1 {
				m.cursor++
			}
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "a":
			// Toggle all.
			allSelected := len(m.selected) == len(m.game.Files)
			m.selected = make(map[int]bool)
			if !allSelected {
				for i := range m.game.Files {
					m.selected[i] = true
				}
			}
		case "enter":
			var files []sites.GameFile
			for i, f := range m.game.Files {
				if m.selected[i] {
					files = append(files, f)
				}
			}
			if len(files) == 0 {
				return m, nil // don't proceed with nothing selected
			}
			return m, func() tea.Msg {
				return pickDoneMsg{game: m.game, files: files}
			}
		}
	}

	return m, nil
}

func (m pickModel) View() string {
	var b strings.Builder

	title := fmt.Sprintf("%s — %s", m.game.Title, m.game.Source)
	b.WriteString(titleStyle.Render(title) + "\n")
	b.WriteString(dimStyle.Render("Select files to download:") + "\n\n")

	for i, f := range m.game.Files {
		cursor := "  "
		if i == m.cursor {
			cursor = accentStyle.Render("> ")
		}

		check := "[ ] "
		if m.selected[i] {
			check = successStyle.Render("[x] ")
		}

		name := f.Name
		if f.Size > 0 {
			name += dimStyle.Render(fmt.Sprintf(" (%s)", formatSize(f.Size)))
		}

		if i == m.cursor {
			name = lipgloss.NewStyle().Foreground(colorText).Render(name)
		}

		b.WriteString(cursor + check + name + "\n")
	}

	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	b.WriteString(fmt.Sprintf("\n%s selected\n", accentStyle.Render(fmt.Sprintf("%d", count))))
	b.WriteString("\n" + helpStyle.Render("space: toggle  a: all  enter: download  ctrl+c: quit"))

	return b.String()
}
