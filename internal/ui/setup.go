package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/cardinal9985/flashgrab/internal/config"
)

type setupStep int

const (
	stepDownloadDir setupStep = iota
	stepItchioKey
	stepConfirm
)

type setupModel struct {
	step       setupStep
	dirInput   textinput.Model
	keyInput   textinput.Model
	err        string
	title      string
	width      int
}

func newSetupModel(cfg *config.Config) setupModel {
	dir := textinput.New()
	dir.Placeholder = cfg.DownloadDir
	dir.SetValue(cfg.DownloadDir)
	dir.Focus()
	dir.CharLimit = 500
	dir.Width = 50

	key := textinput.New()
	key.Placeholder = "paste your API key here (or leave blank to skip)"
	key.SetValue(cfg.Itchio.APIKey)
	key.CharLimit = 200
	key.Width = 50
	key.EchoMode = textinput.EchoPassword

	title := "Settings"
	if !config.Exists() {
		title = "First-time Setup"
	}

	return setupModel{
		step:     stepDownloadDir,
		dirInput: dir,
		keyInput: key,
		title:    title,
	}
}

type setupDoneMsg struct {
	cfg *config.Config
}

func (m setupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m setupModel) Update(msg tea.Msg) (setupModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		m.err = ""

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "enter":
			switch m.step {
			case stepDownloadDir:
				expanded, err := config.ValidateDir(m.dirInput.Value())
				if err != nil {
					m.err = err.Error()
					return m, nil
				}
				m.dirInput.SetValue(expanded)
				m.step = stepItchioKey
				m.dirInput.Blur()
				m.keyInput.Focus()
				return m, textinput.Blink

			case stepItchioKey:
				m.step = stepConfirm
				m.keyInput.Blur()
				return m, nil

			case stepConfirm:
				cfg := &config.Config{
					DownloadDir: m.dirInput.Value(),
					Itchio: config.ItchioConfig{
						APIKey: strings.TrimSpace(m.keyInput.Value()),
					},
				}
				return m, func() tea.Msg {
					return setupDoneMsg{cfg: cfg}
				}
			}

		case "tab", "shift+tab":
			switch m.step {
			case stepDownloadDir:
				m.step = stepItchioKey
				m.dirInput.Blur()
				m.keyInput.Focus()
				return m, textinput.Blink
			case stepItchioKey:
				m.step = stepDownloadDir
				m.keyInput.Blur()
				m.dirInput.Focus()
				return m, textinput.Blink
			}
		}
	}

	var cmd tea.Cmd
	switch m.step {
	case stepDownloadDir:
		m.dirInput, cmd = m.dirInput.Update(msg)
	case stepItchioKey:
		m.keyInput, cmd = m.keyInput.Update(msg)
	}

	return m, cmd
}

func (m setupModel) View() string {
	var b strings.Builder

	header := titleStyle.Render(m.title)
	b.WriteString(header + "\n\n")

	label1 := accentStyle.Render("Download directory")
	if m.step == stepDownloadDir {
		label1 = titleStyle.Render("> Download directory")
	}
	b.WriteString(label1 + "\n")
	b.WriteString("  " + m.dirInput.View() + "\n\n")

	label2 := accentStyle.Render("itch.io API key")
	if m.step == stepItchioKey {
		label2 = titleStyle.Render("> itch.io API key")
	}
	b.WriteString(label2 + "\n")
	b.WriteString(dimStyle.Render("  Get one at: https://itch.io/user/settings/api-keys") + "\n")
	b.WriteString("  " + m.keyInput.View() + "\n\n")

	if m.step == stepConfirm {
		summary := boxStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				accentStyle.Render("Directory: ")+m.dirInput.Value(),
				accentStyle.Render("itch.io:   ")+keyPreview(m.keyInput.Value()),
			),
		)
		b.WriteString(summary + "\n\n")
		b.WriteString(successStyle.Render("Press enter to save") + " " +
			dimStyle.Render("or tab to go back"))
	}

	if m.err != "" {
		b.WriteString("\n" + errorStyle.Render(m.err))
	}

	b.WriteString("\n\n" + helpStyle.Render("tab: next field  enter: confirm  ctrl+c: quit"))

	return b.String()
}

func keyPreview(key string) string {
	if key == "" {
		return dimStyle.Render("(skipped)")
	}
	// Show only the first 4 and last 4 chars.
	if len(key) > 12 {
		return key[:4] + "..." + key[len(key)-4:]
	}
	return "****"
}
