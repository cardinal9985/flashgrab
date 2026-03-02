package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cardinal9985/flashgrab/internal/download"
	"github.com/cardinal9985/flashgrab/internal/sites"
)

type progressModel struct {
	game        *sites.Game
	files       []sites.GameFile
	dlManager   *download.Manager
	current     int
	results     []*download.Result
	errors      []error
	bar         progress.Model
	spin        spinner.Model
	downloaded  int64
	totalSize   int64
	done        bool
	width       int
}

func newProgressModel(game *sites.Game, files []sites.GameFile, mgr *download.Manager) progressModel {
	bar := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
	)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = accentStyle

	return progressModel{
		game:      game,
		files:     files,
		dlManager: mgr,
		bar:       bar,
		spin:      sp,
	}
}

// progressTickMsg carries real-time download progress from the background goroutine.
type progressTickMsg struct {
	downloaded int64
	total      int64
}

// fileCompleteMsg signals that one file finished downloading.
type fileCompleteMsg struct {
	result *download.Result
	err    error
}

// allDoneMsg is sent when every file in the queue has been processed.
type allDoneMsg struct {
	game    *sites.Game
	results []*download.Result
	errors  []error
}

func (m progressModel) Init() tea.Cmd {
	return tea.Batch(
		m.spin.Tick,
		m.downloadNext(),
	)
}

func (m progressModel) downloadNext() tea.Cmd {
	if m.current >= len(m.files) {
		return func() tea.Msg {
			return allDoneMsg{
				game:    m.game,
				results: m.results,
				errors:  m.errors,
			}
		}
	}

	f := m.files[m.current]
	mgr := m.dlManager

	return func() tea.Msg {
		result, err := mgr.Fetch(f.URL, f.Filename, func(dl, total int64) {
			// We can't send tea.Msg from here directly, but bubbletea
			// will pick up the final state when the command returns.
		})
		return fileCompleteMsg{result: result, err: err}
	}
}

func (m progressModel) Update(msg tea.Msg) (progressModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.bar.Width = min(msg.Width-10, 50)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case fileCompleteMsg:
		if msg.err != nil {
			m.errors = append(m.errors, fmt.Errorf("%s: %w", m.files[m.current].Name, msg.err))
		} else {
			m.results = append(m.results, msg.result)
		}
		m.current++
		return m, m.downloadNext()

	case allDoneMsg:
		m.done = true
		return m, func() tea.Msg { return msg }
	}

	return m, nil
}

func (m progressModel) View() string {
	var b strings.Builder

	title := fmt.Sprintf("Downloading from %s", m.game.Source)
	b.WriteString(titleStyle.Render(title) + "\n\n")

	// Show completed files.
	for i, r := range m.results {
		status := successStyle.Render("done")
		if r.Existed {
			status = warnStyle.Render("exists")
		}
		b.WriteString(fmt.Sprintf("  %s %s %s\n",
			status,
			r.Filename,
			dimStyle.Render(formatSize(r.Size)),
		))
		_ = i
	}

	// Show errors.
	for _, e := range m.errors {
		b.WriteString(fmt.Sprintf("  %s %s\n", errorStyle.Render("fail"), e.Error()))
	}

	// Show the currently downloading file.
	if !m.done && m.current < len(m.files) {
		f := m.files[m.current]
		b.WriteString(fmt.Sprintf("\n  %s %s\n", m.spin.View(), f.Name))
	}

	// Progress summary.
	total := len(m.files)
	finished := len(m.results) + len(m.errors)
	if total > 0 {
		pct := float64(finished) / float64(total)
		b.WriteString("\n  " + m.bar.ViewAs(pct) + "\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d / %d files", finished, total)) + "\n")
	}

	return b.String()
}
