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

var (
	// Newgrounds embeds game data as JSON inside an embedController() call.
	ngEmbedStart   = regexp.MustCompile(`embedController\(`)
	ngTitlePattern = regexp.MustCompile(`<title>([^<]+)</title>`)
	ngIDPattern    = regexp.MustCompile(`/view/(\d+)`)
)

// ngEmbed is the JSON structure Newgrounds uses inside embedController().
type ngEmbed struct {
	URL         string `json:"url"`
	Description string `json:"description"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Filesize    int64  `json:"filesize"`
}

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
	idMatch := ngIDPattern.FindStringSubmatch(rawURL)

	resp, err := ng.client.Get(rawURL)
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

	title := ""
	if m := ngTitlePattern.FindStringSubmatch(page); len(m) > 1 {
		title = m[1]
		for _, suffix := range []string{" - Newgrounds.com", " | Newgrounds.com"} {
			title = strings.TrimSuffix(title, suffix)
		}
		title = strings.TrimSpace(title)
	}

	if title == "" && len(idMatch) > 1 {
		title = "ng_" + idMatch[1]
	} else if title == "" {
		title = "untitled"
	}

	embeds, err := ng.parseEmbeds(page)
	if err != nil || len(embeds) == 0 {
		return nil, fmt.Errorf("couldn't find a download link on that page")
	}

	var files []GameFile
	for _, e := range embeds {
		if e.URL == "" {
			continue
		}

		fileURL := strings.ReplaceAll(e.URL, "\\/", "/")

		ext := path.Ext(fileURL)
		if ext == "" {
			ext = guessExtension(e.Description)
		}

		name := title
		if len(embeds) > 1 && e.Description != "" {
			name = fmt.Sprintf("%s (%s)", title, e.Description)
		}

		filename := sanitize.Filename(name, ext)

		files = append(files, GameFile{
			Name:     fmt.Sprintf("%s%s", name, ext),
			URL:      fileURL,
			Size:     e.Filesize,
			Filename: filename,
		})
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("couldn't find a download link on that page")
	}

	return &Game{
		Title:  title,
		Source: "Newgrounds",
		URL:    rawURL,
		Files:  files,
	}, nil
}

// parseEmbeds pulls game file info out of the embedController() call on the page.
// The array often contains raw JS (callback:function(){...}) mixed in with the
// JSON, so we can't just regex out the array and unmarshal it directly.
func (ng *newgrounds) parseEmbeds(page string) ([]ngEmbed, error) {
	loc := ngEmbedStart.FindStringIndex(page)
	if loc == nil {
		return nil, fmt.Errorf("no embed data found")
	}

	start := loc[1]
	for start < len(page) && page[start] != '[' {
		start++
	}
	if start >= len(page) {
		return nil, fmt.Errorf("no embed array found")
	}

	depth := 0
	inString := false
	escaped := false
	end := -1
	for i := start; i < len(page); i++ {
		c := page[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if c == '[' {
			depth++
		} else if c == ']' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}

	if end == -1 {
		return nil, fmt.Errorf("unterminated embed array")
	}

	raw := page[start:end]
	raw = stripJSFunctions(raw)

	var embeds []ngEmbed
	if err := json.Unmarshal([]byte(raw), &embeds); err != nil {
		return nil, fmt.Errorf("parsing embed JSON: %w", err)
	}

	return embeds, nil
}

// stripJSFunctions strips unquoted key:value pairs (like callback:function(){...})
// from a JS object literal so the rest can be parsed as JSON.
func stripJSFunctions(s string) string {
	re := regexp.MustCompile(`,\s*[a-zA-Z_]\w*\s*:`)
	result := s
	for {
		loc := re.FindStringIndex(result)
		if loc == nil {
			break
		}
		valueStart := loc[1]
		depth := 0
		inStr := false
		esc := false
		i := valueStart
		for i < len(result) {
			c := result[i]
			if esc {
				esc = false
				i++
				continue
			}
			if c == '\\' && inStr {
				esc = true
				i++
				continue
			}
			if c == '"' {
				inStr = !inStr
				i++
				continue
			}
			if inStr {
				i++
				continue
			}
			if c == '{' || c == '(' || c == '[' {
				depth++
			} else if c == '}' || c == ')' || c == ']' {
				if depth == 0 {
					break
				}
				depth--
			} else if c == ',' && depth == 0 {
				break
			}
			i++
		}
		result = result[:loc[0]] + result[i:]
	}
	return result
}

// guessExtension falls back to the embed description when the URL has no extension.
func guessExtension(desc string) string {
	lower := strings.ToLower(desc)
	switch {
	case strings.Contains(lower, "flash"):
		return ".swf"
	case strings.Contains(lower, "html"):
		return ".zip"
	default:
		return ".swf"
	}
}
