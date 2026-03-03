package sites

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/cardinal9985/flashgrab/internal/sanitize"
)

type internetArchive struct {
	client *http.Client
}

func init() {
	Register(&internetArchive{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: safeRedirectPolicy,
		},
	})
}

func (ia *internetArchive) Name() string { return "Internet Archive" }

func (ia *internetArchive) Match(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	return host == "archive.org" || host == "www.archive.org"
}

// Resolve handles /details/<id> (item pages) and /download/<id>/<file> (direct links).
func (ia *internetArchive) Resolve(rawURL string) (*Game, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("can't parse archive.org URL—expected /details/ID or /download/ID/file")
	}

	switch parts[0] {
	case "download":
		return ia.resolveDirectLink(rawURL, parts)
	case "details":
		return ia.resolveItemPage(parts[1])
	default:
		return nil, fmt.Errorf("unsupported archive.org path: /%s", parts[0])
	}
}

func (ia *internetArchive) resolveDirectLink(rawURL string, parts []string) (*Game, error) {
	filename := "download"
	if len(parts) >= 3 {
		filename = parts[len(parts)-1]
	}

	ext := path.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	return &Game{
		Title:  name,
		Source: "Internet Archive",
		URL:    rawURL,
		Files: []GameFile{{
			Name:     filename,
			URL:      rawURL,
			Filename: sanitize.Filename(name, ext),
		}},
	}, nil
}

// resolveItemPage fetches the Archive metadata API and filters for game files.
func (ia *internetArchive) resolveItemPage(itemID string) (*Game, error) {
	metaURL := fmt.Sprintf("https://archive.org/metadata/%s", url.PathEscape(itemID))

	resp, err := ia.client.Get(metaURL)
	if err != nil {
		return nil, fmt.Errorf("fetching metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata API returned %d", resp.StatusCode)
	}

	var meta struct {
		Metadata struct {
			Title string `json:"title"`
		} `json:"metadata"`
		Files []struct {
			Name   string `json:"name"`
			Size   string `json:"size"`
			Format string `json:"format"`
		} `json:"files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("parsing metadata: %w", err)
	}

	title := meta.Metadata.Title
	if title == "" {
		title = itemID
	}

	gameExts := map[string]bool{
		".swf": true, ".zip": true, ".7z": true,
		".html": true, ".htm": true, ".exe": true,
	}

	var files []GameFile
	for _, f := range meta.Files {
		ext := strings.ToLower(path.Ext(f.Name))
		if !gameExts[ext] {
			continue
		}

		dlURL := fmt.Sprintf("https://archive.org/download/%s/%s",
			url.PathEscape(itemID), url.PathEscape(f.Name))

		name := strings.TrimSuffix(f.Name, ext)
		files = append(files, GameFile{
			Name:     f.Name,
			URL:      dlURL,
			Filename: sanitize.Filename(name, ext),
		})
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no downloadable game files found in item %q", itemID)
	}

	return &Game{
		Title:  title,
		Source: "Internet Archive",
		URL:    fmt.Sprintf("https://archive.org/details/%s", itemID),
		Files:  files,
	}, nil
}
