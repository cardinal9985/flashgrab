package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorOrange = lipgloss.Color("#FF6600")
	colorGreen  = lipgloss.Color("#04B575")
	colorPurple = lipgloss.Color("#7B2FBE")
	colorDim    = lipgloss.Color("#666666")
	colorText   = lipgloss.Color("#FAFAFA")
	colorRed    = lipgloss.Color("#FF4444")
	colorYellow = lipgloss.Color("#FFB627")
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(colorOrange).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	accentStyle = lipgloss.NewStyle().
			Foreground(colorPurple)

	successStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed)

	warnStyle = lipgloss.NewStyle().
			Foreground(colorYellow)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorOrange).
			Padding(0, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			Italic(true)
)

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
