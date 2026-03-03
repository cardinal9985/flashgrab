package sites

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/cardinal9985/flashgrab/internal/sanitize"
)

var (
	// Old-style direct SWF link.
	kgSWFPattern = regexp.MustCompile(`https?://chat\.kongregate\.com/game_files/[^"'\s]+\.swf`)
	// New-style HTML5 game hosted on konggames.com CDN.
	kgCDNPattern   = regexp.MustCompile(`https?://game\d+\.konggames\.com/game[sz]/[^"'\s]+`)
	kgTitlePattern = regexp.MustCompile(`<title>([^<]+)</title>`)
	kgSlugPattern  = regexp.MustCompile(`/games/[^/]+/([^/?#]+)`)
	kgEmbedPattern = regexp.MustCompile(`/games/([^/]+/[^/]+)/embed`)
)

type kongregate struct {
	client *http.Client
}

func init() {
	Register(&kongregate{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: safeRedirectPolicy,
		},
	})
}

func (kg *kongregate) Name() string { return "Kongregate" }

func (kg *kongregate) Match(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	return host == "www.kongregate.com" || host == "kongregate.com"
}

func (kg *kongregate) Resolve(rawURL string) (*Game, error) {
	page, err := kg.fetchPage(rawURL)
	if err != nil {
		return nil, err
	}

	title := kg.extractTitle(page, rawURL)

	if swf := kgSWFPattern.FindString(page); swf != "" {
		filename := sanitize.Filename(title, ".swf")
		return &Game{
			Title:  title,
			Source: "Kongregate",
			URL:    rawURL,
			Files: []GameFile{{
				Name:     filename,
				URL:      swf,
				Filename: filename,
			}},
		}, nil
	}

	// New-style: game is on the konggames CDN. Check the main page first,
	// then fall back to the /embed page where the URL is usually exposed.
	if cdnURL := kg.findCDNURL(page); cdnURL != "" {
		return kg.buildCDNGame(title, rawURL, cdnURL), nil
	}

	// Try the embed page.
	embedURL := kg.embedURL(rawURL)
	if embedURL != "" {
		if embedPage, err := kg.fetchPage(embedURL); err == nil {
			if cdnURL := kg.findCDNURL(embedPage); cdnURL != "" {
				return kg.buildCDNGame(title, rawURL, cdnURL), nil
			}
		}
	}

	return nil, fmt.Errorf("couldn't find a game file on that page")
}

func (kg *kongregate) fetchPage(rawURL string) (string, error) {
	resp, err := kg.client.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("fetching page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", fmt.Errorf("reading page: %w", err)
	}
	return string(body), nil
}

func (kg *kongregate) extractTitle(page, rawURL string) string {
	if m := kgTitlePattern.FindStringSubmatch(page); len(m) > 1 {
		title := m[1]
		if idx := strings.LastIndex(title, "|"); idx > 0 {
			title = strings.TrimSpace(title[:idx])
		} else {
			title = strings.TrimSpace(title)
		}
		// Kongregate appends " Free Game" or " Free Online Game" to titles.
		for _, suffix := range []string{" Free Online Game", " Free Game"} {
			title = strings.TrimSuffix(title, suffix)
		}
		return title
	}
	if m := kgSlugPattern.FindStringSubmatch(rawURL); len(m) > 1 {
		return strings.ReplaceAll(m[1], "-", " ")
	}
	return "untitled"
}

func (kg *kongregate) findCDNURL(page string) string {
	m := kgCDNPattern.FindString(page)
	if m == "" {
		return ""
	}
	// Strip query params and trailing quotes.
	if idx := strings.IndexAny(m, `"'`); idx >= 0 {
		m = m[:idx]
	}
	return m
}

func (kg *kongregate) embedURL(rawURL string) string {
	// Turn /games/author/slug into /games/author/slug/embed
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	p := strings.TrimSuffix(u.Path, "/")
	// Strip /en/ or other locale prefixes to get /games/author/slug
	if m := kgSlugPattern.FindString(p); m != "" {
		u.Path = m + "/embed"
		return u.String()
	}
	return ""
}

func (kg *kongregate) buildCDNGame(title, rawURL, cdnURL string) *Game {
	ext := ".zip"
	if strings.HasSuffix(cdnURL, ".swf") {
		ext = ".swf"
	}
	filename := sanitize.Filename(title, ext)

	return &Game{
		Title:  title,
		Source: "Kongregate",
		URL:    rawURL,
		Files: []GameFile{{
			Name:     filename,
			URL:      cdnURL,
			Filename: filename,
		}},
	}
}
