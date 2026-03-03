package sites

import (
	"fmt"
	"net/url"
	"strings"
)

// Game holds metadata about a discovered game and its downloadable files.
type Game struct {
	Title      string
	Source     string           // human-readable site name
	URL        string           // the original URL provided by the user
	Files      []GameFile       // at least one entry
	Flashpoint *FlashpointMatch // nil if no Flashpoint match found
}

// GameFile represents a single downloadable file for a game.
type GameFile struct {
	Name     string // display name (e.g. "Game.swf" or "Windows build")
	URL      string // direct download URL
	Size     int64  // content length in bytes, 0 if unknown
	Filename string // suggested filename on disk
}

// Site knows how to detect and resolve game downloads from a particular host.
type Site interface {
	// Name returns a short human-readable label like "Newgrounds".
	Name() string

	// Match reports whether the given URL belongs to this site.
	Match(u *url.URL) bool

	// Resolve fetches game metadata and download links for the given URL.
	Resolve(rawURL string) (*Game, error)
}

// registry is the ordered list of supported sites. The first match wins when
// routing a URL, so more specific sites should come before generic ones.
var registry []Site

// Register adds a site handler to the global registry.
func Register(s Site) {
	registry = append(registry, s)
}

// Resolve routes a URL to the appropriate site handler, resolves the game,
// and returns the result. Returns an error if no site matches.
func Resolve(rawURL string) (*Game, error) {
	rawURL = strings.TrimSpace(rawURL)

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	for _, s := range registry {
		if s.Match(u) {
			return s.Resolve(rawURL)
		}
	}

	return nil, fmt.Errorf("no supported site found for %s", u.Host)
}

// ListSites returns the names of all registered site handlers.
func ListSites() []string {
	names := make([]string, len(registry))
	for i, s := range registry {
		names[i] = s.Name()
	}
	return names
}
