package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cardinal9985/flashgrab/internal/config"
	"github.com/cardinal9985/flashgrab/internal/download"
	"github.com/cardinal9985/flashgrab/internal/sanitize"
	"github.com/cardinal9985/flashgrab/internal/sites"
)

// view represents the current screen in the state machine.
type view int

const (
	viewSetup    view = iota
	viewLogo
	viewInput
	viewResolve
	viewPick
	viewProgress
	viewDone
)

// Model is the top-level bubbletea model that routes between views.
type Model struct {
	view     view
	cfg      *config.Config
	width    int
	height   int
	fpClient *sites.FlashpointClient

	// Sub-models for each view.
	setup    setupModel
	input    inputModel
	resolve  spinner.Model
	pick     pickModel
	progress progressModel
	done     doneModel

	// Stashed state between views.
	resolveURL string
}

// New creates the root TUI model. If firstRun is true, the setup wizard is
// shown before anything else.
func New(cfg *config.Config, firstRun bool) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = accentStyle

	m := Model{
		cfg:      cfg,
		resolve:  sp,
		fpClient: sites.NewFlashpointClient(),
	}

	if firstRun {
		m.view = viewSetup
		m.setup = newSetupModel(cfg)
	} else {
		m.view = viewLogo
	}

	return m
}

// splashDoneMsg is sent after the logo display timeout.
type splashDoneMsg struct{}

func (m Model) Init() tea.Cmd {
	switch m.view {
	case viewSetup:
		return m.setup.Init()
	case viewLogo:
		return tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg {
			return splashDoneMsg{}
		})
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// Any keypress skips the splash screen.
		if m.view == viewLogo {
			return m.switchToInput()
		}
	}

	switch m.view {
	case viewSetup:
		return m.updateSetup(msg)
	case viewLogo:
		return m.updateLogo(msg)
	case viewInput:
		return m.updateInput(msg)
	case viewResolve:
		return m.updateResolve(msg)
	case viewPick:
		return m.updatePick(msg)
	case viewProgress:
		return m.updateProgress(msg)
	case viewDone:
		return m.updateDone(msg)
	}

	return m, nil
}

func (m Model) View() string {
	content := ""

	switch m.view {
	case viewSetup:
		content = m.setup.View()
	case viewLogo:
		content = renderLogo()
	case viewInput:
		content = m.input.View()
	case viewResolve:
		content = titleStyle.Render("Looking up game...") + "\n\n" +
			"  " + m.resolve.View() + " " + dimStyle.Render(m.resolveURL) + "\n"
	case viewPick:
		content = m.pick.View()
	case viewProgress:
		content = m.progress.View()
	case viewDone:
		content = m.done.View()
	}

	// Add consistent padding around everything.
	return lipgloss.NewStyle().Padding(1, 2).Render(content)
}

// -- View transition helpers --

func (m Model) switchToInput() (tea.Model, tea.Cmd) {
	m.view = viewInput
	m.input = newInputModel()
	return m, m.input.Init()
}

func (m Model) updateSetup(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case setupDoneMsg:
		m.cfg = msg.cfg
		if err := config.Save(msg.cfg); err != nil {
			// Not much we can do here. The config just won't persist.
			_ = err
		}
		// Re-register itch.io with the new key.
		sites.NewItchio(m.cfg.Itchio.APIKey)
		m.view = viewLogo
		return m, tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg {
			return splashDoneMsg{}
		})
	}

	var cmd tea.Cmd
	m.setup, cmd = m.setup.Update(msg)
	return m, cmd
}

func (m Model) updateLogo(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(splashDoneMsg); ok {
		return m.switchToInput()
	}
	return m, nil
}

func (m Model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resolveMsg:
		if msg.err != nil {
			m.view = viewInput
			m.input.err = msg.err.Error()
			m.input.textInput.Focus()
			return m, nil
		}

		// Try to improve filenames using Flashpoint.
		m.improveFilenames(msg.game)

		if len(msg.game.Files) == 1 {
			// Skip the pick screen for single-file games.
			mgr := download.New(m.cfg.DownloadDir)
			m.progress = newProgressModel(msg.game, msg.game.Files, mgr)
			m.view = viewProgress
			return m, m.progress.Init()
		}

		m.pick = newPickModel(msg.game)
		m.view = viewPick
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	// If the input triggered a resolve, switch to the spinner view.
	if m.input.textInput.Value() != "" {
		if _, ok := msg.(tea.KeyMsg); ok {
			if msg.(tea.KeyMsg).String() == "enter" && m.input.err == "" {
				url := m.input.textInput.Value()
				if _, err := sanitize.URL(url); err == nil {
					m.view = viewResolve
					m.resolveURL = url
					return m, tea.Batch(cmd, m.resolve.Tick)
				}
			}
		}
	}

	return m, cmd
}

func (m Model) updateResolve(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resolveMsg:
		if msg.err != nil {
			m.view = viewInput
			m.input.err = msg.err.Error()
			m.input.textInput.Focus()
			return m, nil
		}

		m.improveFilenames(msg.game)

		if len(msg.game.Files) == 1 {
			mgr := download.New(m.cfg.DownloadDir)
			m.progress = newProgressModel(msg.game, msg.game.Files, mgr)
			m.view = viewProgress
			return m, m.progress.Init()
		}

		m.pick = newPickModel(msg.game)
		m.view = viewPick
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.resolve, cmd = m.resolve.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) updatePick(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pickDoneMsg:
		mgr := download.New(m.cfg.DownloadDir)
		m.progress = newProgressModel(msg.game, msg.files, mgr)
		m.view = viewProgress
		return m, m.progress.Init()
	}

	var cmd tea.Cmd
	m.pick, cmd = m.pick.Update(msg)
	return m, cmd
}

func (m Model) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case allDoneMsg:
		m.done = newDoneModel(msg.game, msg.results, msg.errors)
		m.view = viewDone
		return m, nil
	}

	var cmd tea.Cmd
	m.progress, cmd = m.progress.Update(msg)
	return m, cmd
}

func (m Model) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case restartMsg:
		return m.switchToInput()
	}

	var cmd tea.Cmd
	m.done, cmd = m.done.Update(msg)
	return m, cmd
}

// improveFilenames queries Flashpoint for each file and updates the filename
// to match the canonical title if a match is found.
func (m Model) improveFilenames(game *sites.Game) {
	canonical := m.fpClient.LookupTitle(game.Title)
	if canonical == "" || canonical == game.Title {
		return
	}

	game.Title = canonical
	for i := range game.Files {
		ext := ""
		name := game.Files[i].Filename
		if dot := lastDot(name); dot >= 0 {
			ext = name[dot:]
			name = name[:dot]
		}
		game.Files[i].Filename = sanitize.Filename(canonical, ext)
	}
}

func lastDot(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}
