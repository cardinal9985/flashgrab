package ui

import "github.com/charmbracelet/lipgloss"

const logoArt = `
 _____ _           _     ____           _
|  ___| | __ _ ___| |__ / ___|_ __ __ _| |__
| |_  | |/ _` + "`" + ` / __| '_ \| |  _| '__/ _` + "`" + ` | '_ \
|  _| | | (_| \__ \ | | | |_| | | | (_| | |_) |
|_|   |_|\__,_|___/_| |_|\____|_|  \__,_|_.__/`

var logoStyle = lipgloss.NewStyle().
	Foreground(colorOrange).
	Bold(true)

var taglineStyle = lipgloss.NewStyle().
	Foreground(colorDim).
	Italic(true)

func renderLogo() string {
	logo := logoStyle.Render(logoArt)
	tagline := taglineStyle.Render("  grab flash games from the web")
	return logo + "\n" + tagline + "\n"
}
