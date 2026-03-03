package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cardinal9985/flashgrab/internal/sanitize"
	"github.com/cardinal9985/flashgrab/internal/sites"
)

type inputModel struct {
	textInput textinput.Model
	err       string
	width     int
}

func newInputModel() inputModel {
	ti := textinput.New()
	ti.Placeholder = "paste a URL here..."
	ti.Focus()
	ti.CharLimit = 2048
	ti.Width = 60

	return inputModel{textInput: ti}
}

type resolveMsg struct {
	game *sites.Game
	err  error
}

type openSettingsMsg struct{}

func (m inputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputModel) Update(msg tea.Msg) (inputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		m.err = ""

		switch msg.String() {
		case "ctrl+s":
			return m, func() tea.Msg { return openSettingsMsg{} }
		case "enter":
			url := strings.TrimSpace(m.textInput.Value())
			if url == "" {
				return m, nil
			}

			if _, err := sanitize.URL(url); err != nil {
				m.err = err.Error()
				return m, nil
			}

			return m, func() tea.Msg {
				game, err := sites.Resolve(url)
				return resolveMsg{game: game, err: err}
			}
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Enter a game URL") + "\n")
	b.WriteString(dimStyle.Render("Supports: Newgrounds, itch.io, Kongregate, Internet Archive") + "\n\n")
	b.WriteString("  " + m.textInput.View() + "\n")

	if m.err != "" {
		b.WriteString("\n" + errorStyle.Render("  "+m.err) + "\n")
	}

	b.WriteString("\n" + helpStyle.Render("enter: search  ctrl+s: settings  ctrl+c: quit"))

	return b.String()
}
