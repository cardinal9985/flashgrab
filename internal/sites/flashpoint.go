package sites

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FlashpointClient queries the Flashpoint Archive database to find canonical
// game titles. This isn't a Site implementation—it's a supplementary lookup
// used by the download pipeline to improve filenames.
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

// LookupTitle searches Flashpoint for a game by title and returns the
// canonical title if found. Returns an empty string (not an error) when
// nothing matches—this is expected for games that aren't in the archive.
func (fp *FlashpointClient) LookupTitle(title string) string {
	if title == "" {
		return ""
	}

	// Use the search endpoint with the title as the query.
	params := url.Values{
		"search": {title},
	}
	searchURL := fp.baseURL + "/search?" + params.Encode()

	resp, err := fp.client.Get(searchURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var results []struct {
		Title string `json:"title"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return ""
	}

	if len(results) == 0 {
		return ""
	}

	// Look for an exact or near-exact match. Flashpoint search is fuzzy,
	// so we verify that the top result is actually close to what we asked for.
	for _, r := range results {
		if strings.EqualFold(r.Title, title) {
			return r.Title
		}
	}

	// If nothing matched exactly, check if the top result contains our query
	// as a substring. This catches cases like "Game Name v1.2" matching "Game Name".
	top := results[0].Title
	if strings.Contains(strings.ToLower(top), strings.ToLower(title)) ||
		strings.Contains(strings.ToLower(title), strings.ToLower(top)) {
		return top
	}

	return ""
}
