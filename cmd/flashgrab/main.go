package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cardinal9985/flashgrab/internal/config"
	"github.com/cardinal9985/flashgrab/internal/download"
	"github.com/cardinal9985/flashgrab/internal/sanitize"
	"github.com/cardinal9985/flashgrab/internal/sites"
	"github.com/cardinal9985/flashgrab/internal/ui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("flashgrab %s\n", version)
			return
		case "--help", "-h":
			printUsage()
			return
		case "config":
			runSetup()
			return
		default:
			runCLI(os.Args[1])
			return
		}
	}

	runTUI()
}

func printUsage() {
	fmt.Printf(`flashgrab %s — grab flash games from the web

Usage:
  flashgrab              launch interactive TUI
  flashgrab <url>        download a game directly (no TUI)
  flashgrab config       re-run the setup wizard
  flashgrab --version    print version
  flashgrab --help       show this message

Supported sites:
  Newgrounds             https://www.newgrounds.com/portal/view/...
  itch.io                https://author.itch.io/game (requires API key)
  Kongregate             https://www.kongregate.com/games/author/game
  Internet Archive       https://archive.org/details/... or /download/...
`, version)
}

func runTUI() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %s\n", err)
		os.Exit(1)
	}

	firstRun := !config.Exists()

	// Register itch.io with whatever key we have (empty is fine, it'll
	// just show an error if someone tries to use it without one).
	sites.NewItchio(cfg.Itchio.APIKey)

	m := ui.New(cfg, firstRun)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func runSetup() {
	cfg, _ := config.Load()

	firstRun := true
	sites.NewItchio(cfg.Itchio.APIKey)

	m := ui.New(cfg, firstRun)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

// runCLI handles the non-interactive single-URL download mode. Useful for
// scripting and piping.
func runCLI(rawURL string) {
	if _, err := sanitize.URL(rawURL); err != nil {
		fmt.Fprintf(os.Stderr, "invalid URL: %s\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %s\n", err)
		os.Exit(1)
	}

	sites.NewItchio(cfg.Itchio.APIKey)

	fmt.Printf("Resolving %s...\n", rawURL)

	game, err := sites.Resolve(rawURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	fp := sites.NewFlashpointClient()
	if match := fp.Lookup(game.Title, rawURL); match != nil {
		game.Flashpoint = match
		if match.Title != "" && match.Title != game.Title {
			game.Title = match.Title
			for i := range game.Files {
				ext := ""
				name := game.Files[i].Filename
				if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
					ext = name[dot:]
				}
				game.Files[i].Filename = sanitize.Filename(match.Title, ext)
			}
		}
	}

	fmt.Printf("Found: %s (%s) — %d file(s)\n", game.Title, game.Source, len(game.Files))
	if game.Flashpoint != nil {
		info := fmt.Sprintf("Flashpoint: %q", game.Flashpoint.Title)
		if game.Flashpoint.Developer != "" {
			info += " by " + game.Flashpoint.Developer
		}
		if game.Flashpoint.Platform != "" {
			info += " (" + game.Flashpoint.Platform + ")"
		}
		fmt.Println(info)
	}

	mgr := download.New(cfg.DownloadDir)

	exitCode := 0
	for _, f := range game.Files {
		fmt.Printf("Downloading %s... ", f.Filename)

		result, err := mgr.Fetch(f.URL, f.Filename, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed: %s\n", err)
			exitCode = 1
			continue
		}

		if result.Existed {
			fmt.Printf("already exists\n")
		} else {
			fmt.Printf("saved (%s)\n", formatSizeCLI(result.Size))
		}
	}

	os.Exit(exitCode)
}

func formatSizeCLI(bytes int64) string {
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
