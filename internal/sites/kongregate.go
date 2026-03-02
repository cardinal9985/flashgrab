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
	kgSWFPattern   = regexp.MustCompile(`https?://chat\.kongregate\.com/game_files/[^"'\s]+\.swf`)
	kgTitlePattern = regexp.MustCompile(`<title>([^<]+)</title>`)
	kgSlugPattern  = regexp.MustCompile(`/games/[^/]+/([^/?#]+)`)
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
	resp, err := kg.client.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetching page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("reading page: %w", err)
	}
	page := string(body)

	swfMatch := kgSWFPattern.FindString(page)
	if swfMatch == "" {
		return nil, fmt.Errorf("couldn't find a .swf link on that page")
	}

	// Try <title> first, then fall back to the URL slug.
	title := ""
	if m := kgTitlePattern.FindStringSubmatch(page); len(m) > 1 {
		title = m[1]
		// Kongregate titles often end with " | Kongregate" or similar.
		if idx := strings.LastIndex(title, "|"); idx > 0 {
			title = strings.TrimSpace(title[:idx])
		}
	}

	if title == "" {
		if m := kgSlugPattern.FindStringSubmatch(rawURL); len(m) > 1 {
			title = strings.ReplaceAll(m[1], "-", " ")
		}
	}

	if title == "" {
		title = "untitled"
	}

	filename := sanitize.Filename(title, ".swf")

	return &Game{
		Title:  title,
		Source: "Kongregate",
		URL:    rawURL,
		Files: []GameFile{{
			Name:     filename,
			URL:      swfMatch,
			Filename: filename,
		}},
	}, nil
}
