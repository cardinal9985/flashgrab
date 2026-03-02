package sites

import (
	"encoding/json"
	"fmt"
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

// NewItchio creates an itch.io site handler. Pass an empty apiKey to disable
// itch.io support entirely; Resolve will return a helpful error.
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

	// Parse the game URL to extract author and slug.
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	// Look up the game ID through the itch.io page.
	gameID, title, err := it.lookupGame(rawURL, u)
	if err != nil {
		return nil, err
	}

	// Fetch the list of uploads for this game.
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
// doesn't provide a way to look up games by URL, so we have to do this the
// messy way.
func (it *itchio) lookupGame(rawURL string, u *url.URL) (int, string, error) {
	resp, err := it.client.Get(rawURL)
	if err != nil {
		return 0, "", fmt.Errorf("fetching game page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("game page returned status %d", resp.StatusCode)
	}

	// itch.io embeds JSON-LD and data attributes with the game ID.
	// Look for data-game_id or the JSON object.
	var buf [512 * 1024]byte
	n, _ := resp.Body.Read(buf[:])
	page := string(buf[:n])

	// Try the data attribute first.
	gameIDPattern := regexp.MustCompile(`data-game_id="(\d+)"`)
	if m := gameIDPattern.FindStringSubmatch(page); len(m) > 1 {
		var id int
		fmt.Sscanf(m[1], "%d", &id)

		title := extractItchTitle(page, u)
		return id, title, nil
	}

	// Fall back to the JSON-LD.
	gameIDJSON := regexp.MustCompile(`"game_id"\s*:\s*(\d+)`)
	if m := gameIDJSON.FindStringSubmatch(page); len(m) > 1 {
		var id int
		fmt.Sscanf(m[1], "%d", &id)

		title := extractItchTitle(page, u)
		return id, title, nil
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
		Uploads []itchUpload `json:"uploads"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing uploads: %w", err)
	}

	// Fill in display names from filenames if missing.
	for i := range result.Uploads {
		if result.Uploads[i].DisplayName == "" {
			result.Uploads[i].DisplayName = result.Uploads[i].Filename
		}
	}

	return result.Uploads, nil
}

func (it *itchio) getDownloadURL(uploadID int) (string, error) {
	apiURL := fmt.Sprintf("https://itch.io/api/1/%s/upload/%d/download", it.apiKey, uploadID)

	// The API returns a JSON object with a URL field. We need to NOT follow
	// the redirect here since we want the signed URL, not the file contents.
	noRedirect := *it.client
	noRedirect.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := noRedirect.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("requesting download URL: %w", err)
	}
	defer resp.Body.Close()

	// The API might return a redirect directly or a JSON body.
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
