package sites

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
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

	// If nothing came back, try again with version junk stripped
	// ("Interactive Buddy v.1.01" → "Interactive Buddy").
	if len(results) == 0 {
		clean := stripVersion(title)
		if clean != title {
			results = fp.query(clean)
		}
	}

	if len(results) == 0 {
		return nil
	}

	return fp.bestMatch(results, title, sourceURL)
}

func (fp *FlashpointClient) query(title string) []flashpointEntry {
	fields := "id,title,developer,source,launchCommand,platform"

	// Try exact title first, fall back to smart search for fuzzy matching
	// (handles version suffixes like "v.1.01" that sites append to titles).
	for _, param := range []string{"title", "smartSearch"} {
		params := url.Values{
			param:    {title},
			"fields": {fields},
			"limit":  {"20"},
		}
		searchURL := fp.baseURL + "/search?" + params.Encode()

		resp, err := fp.client.Get(searchURL)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		var results []flashpointEntry
		if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		if len(results) > 0 {
			return results
		}
	}

	return nil
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

// versionRe matches trailing version strings like "v.1.01", "v2", "Version 1.0",
// "1.02", "(v3.1)" etc. that sites append to game titles.
var versionRe = regexp.MustCompile(`(?i)\s*[\(\[]?\s*v(?:ersion)?\.?\s*\d[\d.]*\s*[\)\]]?\s*$`)

func stripVersion(title string) string {
	cleaned := versionRe.ReplaceAllString(title, "")
	return strings.TrimSpace(cleaned)
}
