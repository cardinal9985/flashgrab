package sites

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FlashpointMatch holds metadata from a confirmed Flashpoint Archive entry.
type FlashpointMatch struct {
	ID        string
	Title     string
	Developer string
	Source    string // original page URL
	Platform  string
}

// FlashpointClient looks up games in the Flashpoint Archive database.
type FlashpointClient struct {
	client  *http.Client
	baseURL string
}

func NewFlashpointClient() *FlashpointClient {
	return &FlashpointClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://db-api.unstable.life",
	}
}

type flashpointEntry struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Developer     string `json:"developer"`
	Source        string `json:"source"`
	LaunchCommand string `json:"launchCommand"`
	Platform      string `json:"platform"`
}

// Lookup searches Flashpoint for a game by title and source URL, returning the
// best match or nil if nothing matched well enough.
func (fp *FlashpointClient) Lookup(title, sourceURL string) *FlashpointMatch {
	if title == "" {
		return nil
	}

	results := fp.query(title)
	if len(results) == 0 {
		return nil
	}

	return fp.bestMatch(results, title, sourceURL)
}

func (fp *FlashpointClient) query(title string) []flashpointEntry {
	params := url.Values{
		"title":  {title},
		"fields": {"id,title,developer,source,launchCommand,platform"},
		"limit":  {"20"},
	}
	searchURL := fp.baseURL + "/search?" + params.Encode()

	resp, err := fp.client.Get(searchURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var results []flashpointEntry
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil
	}

	return results
}

func (fp *FlashpointClient) bestMatch(entries []flashpointEntry, title, sourceURL string) *FlashpointMatch {
	normSource := normalizeURL(sourceURL)

	type scored struct {
		entry flashpointEntry
		score int
	}

	var best *scored

	for _, e := range entries {
		s := 0

		// URL matching (strongest signal)
		if normSource != "" {
			normEntry := normalizeURL(e.Source)
			if normEntry != "" {
				if normEntry == normSource {
					// Exact URL match — return immediately
					return entryToMatch(e)
				}
				if pathsMatch(normSource, normEntry) {
					s = 100
				}
			}
		}

		// Title matching
		if strings.EqualFold(e.Title, title) {
			s += 50
		} else if strings.Contains(strings.ToLower(e.Title), strings.ToLower(title)) ||
			strings.Contains(strings.ToLower(title), strings.ToLower(e.Title)) {
			s += 20
		}

		if s > 0 && (best == nil || s > best.score) {
			best = &scored{entry: e, score: s}
		}
	}

	if best == nil {
		return nil
	}

	return entryToMatch(best.entry)
}

func entryToMatch(e flashpointEntry) *FlashpointMatch {
	return &FlashpointMatch{
		ID:        e.ID,
		Title:     e.Title,
		Developer: e.Developer,
		Source:    e.Source,
		Platform:  e.Platform,
	}
}

// normalizeURL strips scheme, www prefix, and trailing slash for comparison.
func normalizeURL(raw string) string {
	if raw == "" {
		return ""
	}
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "www.")
	s = strings.TrimRight(s, "/")
	return strings.ToLower(s)
}

// pathsMatch checks if two normalized URLs share the same path component
// (e.g. /portal/view/218014), ignoring the domain.
func pathsMatch(a, b string) bool {
	pathA := extractPath(a)
	pathB := extractPath(b)
	return pathA != "" && pathA == pathB
}

func extractPath(normalized string) string {
	idx := strings.IndexByte(normalized, '/')
	if idx < 0 {
		return ""
	}
	return strings.TrimRight(normalized[idx:], "/")
}
