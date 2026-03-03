package ui

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cardinal9985/flashgrab/internal/download"
	"github.com/cardinal9985/flashgrab/internal/sites"
)

// dlProgress is shared between the download goroutine and the UI tick.
type dlProgress struct {
	downloaded atomic.Int64
	total      atomic.Int64
}

type progressModel struct {
	game      *sites.Game
	files     []sites.GameFile
	dlManager *download.Manager
	current   int
	results   []*download.Result
	errors    []error
	bar       progress.Model
	spin      spinner.Model
	progress  *dlProgress // shared with the download goroutine
	fileDL    int64       // last polled value
	fileTotal int64       // last polled value
	done      bool
	width     int
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
		progress:  &dlProgress{},
	}
}

type progressTickMsg struct {
	downloaded int64
	total      int64
}

type fileCompleteMsg struct {
	result *download.Result
	err    error
}

type allDoneMsg struct {
	game    *sites.Game
	results []*download.Result
	errors  []error
}

func (m progressModel) Init() tea.Cmd {
	return tea.Batch(
		m.spin.Tick,
		m.downloadNext(),
		pollProgress(),
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
	p := m.progress

	p.downloaded.Store(0)
	p.total.Store(0)

	return func() tea.Msg {
		result, err := mgr.Fetch(f.URL, f.Filename, func(dl, total int64) {
			p.downloaded.Store(dl)
			p.total.Store(total)
		})
		return fileCompleteMsg{result: result, err: err}
	}
}

func pollProgress() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return progressTickMsg{}
	})
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

	case progressTickMsg:
		if m.done {
			return m, nil
		}
		m.fileDL = m.progress.downloaded.Load()
		m.fileTotal = m.progress.total.Load()
		return m, pollProgress()

	case fileCompleteMsg:
		m.fileDL = 0
		m.fileTotal = 0
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
	for _, r := range m.results {
		status := successStyle.Render("done")
		if r.Existed {
			status = warnStyle.Render("exists")
		}
		b.WriteString(fmt.Sprintf("  %s %s %s\n",
			status,
			r.Filename,
			dimStyle.Render(formatSize(r.Size)),
		))
	}

	// Show errors.
	for _, e := range m.errors {
		b.WriteString(fmt.Sprintf("  %s %s\n", errorStyle.Render("fail"), e.Error()))
	}

	// Show the currently downloading file with byte-level progress.
	if !m.done && m.current < len(m.files) {
		f := m.files[m.current]
		b.WriteString(fmt.Sprintf("\n  %s %s", m.spin.View(), f.Name))
		if m.fileTotal > 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  %s / %s",
				formatSize(m.fileDL), formatSize(m.fileTotal))))
		} else if m.fileDL > 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  %s", formatSize(m.fileDL))))
		}
		b.WriteString("\n")

		// Per-file progress bar.
		if m.fileTotal > 0 {
			pct := float64(m.fileDL) / float64(m.fileTotal)
			b.WriteString("  " + m.bar.ViewAs(pct) + "\n")
		}
	}

	// File count summary.
	total := len(m.files)
	finished := len(m.results) + len(m.errors)
	if total > 1 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("\n  %d / %d files", finished, total)) + "\n")
	}

	return b.String()
}
