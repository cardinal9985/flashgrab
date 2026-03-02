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
	ngSWFPattern   = regexp.MustCompile(`https://uploads\.ungrounded\.net/[^"'\s]+\.(swf|mp4|webm)`)
	ngTitlePattern = regexp.MustCompile(`<title>([^<]+)</title>`)
	ngIDPattern    = regexp.MustCompile(`/view/(\d+)`)
)

type newgrounds struct {
	client *http.Client
}

func init() {
	Register(&newgrounds{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: safeRedirectPolicy,
		},
	})
}

func (ng *newgrounds) Name() string { return "Newgrounds" }

func (ng *newgrounds) Match(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	return host == "www.newgrounds.com" || host == "newgrounds.com"
}

func (ng *newgrounds) Resolve(rawURL string) (*Game, error) {
	// Pull the submission ID for fallback naming.
	idMatch := ngIDPattern.FindStringSubmatch(rawURL)

	resp, err := ng.client.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetching page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	// Read up to 2MB—more than enough for the page head and embed markup.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("reading page: %w", err)
	}
	page := string(body)

	// Find the media URL.
	mediaMatch := ngSWFPattern.FindString(page)
	if mediaMatch == "" {
		return nil, fmt.Errorf("couldn't find a download link on that page")
	}

	// Extract the title from the <title> tag.
	title := ""
	if m := ngTitlePattern.FindStringSubmatch(page); len(m) > 1 {
		title = m[1]
		// Strip the site suffix that Newgrounds appends.
		for _, suffix := range []string{" - Newgrounds.com", " | Newgrounds.com"} {
			title = strings.TrimSuffix(title, suffix)
		}
		title = strings.TrimSpace(title)
	}

	// Fall back to the submission ID if we couldn't get a title.
	if title == "" && len(idMatch) > 1 {
		title = "ng_" + idMatch[1]
	} else if title == "" {
		title = "untitled"
	}

	// Figure out the file extension from the matched URL.
	ext := ".swf"
	if strings.HasSuffix(mediaMatch, ".mp4") {
		ext = ".mp4"
	} else if strings.HasSuffix(mediaMatch, ".webm") {
		ext = ".webm"
	}

	filename := sanitize.Filename(title, ext)

	return &Game{
		Title:  title,
		Source: "Newgrounds",
		URL:    rawURL,
		Files: []GameFile{{
			Name:     filename,
			URL:      mediaMatch,
			Filename: filename,
		}},
	}, nil
}
