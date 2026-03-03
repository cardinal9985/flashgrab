package ui

import "github.com/charmbracelet/lipgloss"

const logoArt = `
 ▛▀▘▌  ▞▀▖▞▀▖▌ ▌▞▀▖▛▀▖▞▀▖▛▀▖
 ▙▄ ▌  ▙▄▌▚▄ ▙▄▌▌▄▖▙▄▘▙▄▌▙▄▘
 ▌  ▌  ▌ ▌▖ ▌▌ ▌▌ ▌▌▚ ▌ ▌▌ ▌
 ▘  ▀▀▘▘ ▘▝▀ ▘ ▘▝▀ ▘ ▘▘ ▘▀▀`

var logoStyle = lipgloss.NewStyle().
	Foreground(colorOrange).
	Bold(true)

var taglineStyle = lipgloss.NewStyle().
	Foreground(colorDim)

func renderLogo() string {
	logo := logoStyle.Render(logoArt)
	tagline := taglineStyle.Render("[ game preservation toolkit ]")

	logoWidth := lipgloss.Width(logo)
	taglineWidth := lipgloss.Width(tagline)
	w := logoWidth
	if taglineWidth > w {
		w = taglineWidth
	}
	centered := lipgloss.NewStyle().Width(w).Align(lipgloss.Center)

	return logo + "\n\n" + centered.Render(tagline) + "\n"
}
