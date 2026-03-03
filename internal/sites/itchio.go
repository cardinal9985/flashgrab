package sites

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/cardinal9985/flashgrab/internal/sanitize"
)

var itchioSlugPattern = regexp.MustCompile(`^([a-zA-Z0-9_-]+)\.itch\.io/([a-zA-Z0-9_-]+)`)

type itchio struct {
	client *http.Client
	apiKey string
}

// NewItchio registers the itch.io handler. An empty apiKey is fine — Resolve
// will just tell the user to set one up.
func NewItchio(apiKey string) {
	Register(&itchio{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: safeRedirectPolicy,
		},
		apiKey: apiKey,
	})
}

func (it *itchio) Name() string { return "itch.io" }

func (it *itchio) Match(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	return strings.HasSuffix(host, ".itch.io") || host == "itch.io"
}

func (it *itchio) Resolve(rawURL string) (*Game, error) {
	if it.apiKey == "" {
		return nil, fmt.Errorf("itch.io requires an API key — run 'flashgrab config' to set one up")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	gameID, title, err := it.lookupGame(rawURL, u)
	if err != nil {
		return nil, err
	}

	uploads, err := it.fetchUploads(gameID)
	if err != nil {
		return nil, err
	}

	if len(uploads) == 0 {
		return nil, fmt.Errorf("no downloadable files found for this game")
	}

	var files []GameFile
	for _, up := range uploads {
		dlURL, err := it.getDownloadURL(up.ID)
		if err != nil {
			continue // skip uploads we can't get a link for
		}

		ext := path.Ext(up.Filename)
		name := strings.TrimSuffix(up.Filename, ext)

		files = append(files, GameFile{
			Name:     up.DisplayName,
			URL:      dlURL,
			Size:     up.Size,
			Filename: sanitize.Filename(name, ext),
		})
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("couldn't get download links for any uploads")
	}

	return &Game{
		Title:  title,
		Source: "itch.io",
		URL:    rawURL,
		Files:  files,
	}, nil
}

type itchUpload struct {
	ID          int    `json:"id"`
	Filename    string `json:"filename"`
	DisplayName string `json:"display_name"`
	Size        int64  `json:"size"`
}

// lookupGame scrapes the itch.io page to find the game ID and title. The API
// doesn't have a URL-to-ID lookup, so we scrape data-game_id from the HTML.
func (it *itchio) lookupGame(rawURL string, u *url.URL) (int, string, error) {
	resp, err := it.client.Get(rawURL)
	if err != nil {
		return 0, "", fmt.Errorf("fetching game page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("game page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return 0, "", fmt.Errorf("reading page: %w", err)
	}
	page := string(body)

	// itch.io puts the game ID in different places depending on the page type.
	// Try them all: data attribute, JSON-LD, and the JS init call.
	idPatterns := []*regexp.Regexp{
		regexp.MustCompile(`data-game_id="(\d+)"`),
		regexp.MustCompile(`"game_id"\s*:\s*(\d+)`),
		regexp.MustCompile(`"game"\s*:\s*\{[^}]*"id"\s*:\s*(\d+)`),
	}

	for _, pat := range idPatterns {
		if m := pat.FindStringSubmatch(page); len(m) > 1 {
			var id int
			fmt.Sscanf(m[1], "%d", &id)
			if id > 0 {
				title := extractItchTitle(page, u)
				return id, title, nil
			}
		}
	}

	return 0, "", fmt.Errorf("couldn't find game ID on the page")
}

func extractItchTitle(page string, u *url.URL) string {
	titlePattern := regexp.MustCompile(`<title>([^<]+)</title>`)
	if m := titlePattern.FindStringSubmatch(page); len(m) > 1 {
		title := m[1]
		// itch.io appends " by AuthorName" to titles.
		if idx := strings.LastIndex(title, " by "); idx > 0 {
			title = title[:idx]
		}
		return strings.TrimSpace(title)
	}

	// Use the URL slug as a last resort.
	slug := strings.Trim(u.Path, "/")
	return strings.ReplaceAll(slug, "-", " ")
}

func (it *itchio) fetchUploads(gameID int) ([]itchUpload, error) {
	apiURL := fmt.Sprintf("https://itch.io/api/1/%s/game/%d/uploads", it.apiKey, gameID)

	resp, err := it.client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("fetching uploads: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("API key was rejected — check your itch.io API key")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("uploads API returned status %d", resp.StatusCode)
	}

	var result struct {
		Uploads json.RawMessage `json:"uploads"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing uploads: %w", err)
	}

	// The itch.io API returns {} instead of [] when there are no uploads.
	var uploads []itchUpload
	if len(result.Uploads) > 0 && result.Uploads[0] == '[' {
		if err := json.Unmarshal(result.Uploads, &uploads); err != nil {
			return nil, fmt.Errorf("parsing uploads array: %w", err)
		}
	}
	// If it starts with '{', it's an empty object — uploads stays nil.

	for i := range uploads {
		if uploads[i].DisplayName == "" {
			uploads[i].DisplayName = uploads[i].Filename
		}
	}

	return uploads, nil
}

func (it *itchio) getDownloadURL(uploadID int) (string, error) {
	apiURL := fmt.Sprintf("https://itch.io/api/1/%s/upload/%d/download", it.apiKey, uploadID)

	// Don't follow the redirect — we want the signed URL, not the file.
	noRedirect := *it.client
	noRedirect.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := noRedirect.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("requesting download URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect {
		loc := resp.Header.Get("Location")
		if loc != "" {
			return loc, nil
		}
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download API returned status %d", resp.StatusCode)
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parsing download URL: %w", err)
	}

	if result.URL == "" {
		return "", fmt.Errorf("no download URL returned")
	}

	return result.URL, nil
}
