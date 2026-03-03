package sites

import "testing"

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://www.newgrounds.com/portal/view/59593", "newgrounds.com/portal/view/59593"},
		{"http://www.newgrounds.com/portal/view/59593/", "newgrounds.com/portal/view/59593"},
		{"https://newgrounds.com/portal/view/59593", "newgrounds.com/portal/view/59593"},
		{"http://example.com", "example.com"},
		{"https://EXAMPLE.COM/Path/", "example.com/path"},
		{"", ""},
		{"  https://example.com/  ", "example.com"},
	}

	for _, tt := range tests {
		got := normalizeURL(tt.input)
		if got != tt.want {
			t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"newgrounds.com/portal/view/59593", "/portal/view/59593"},
		{"example.com", ""},
		{"example.com/", ""},
		{"example.com/a/b/c", "/a/b/c"},
	}

	for _, tt := range tests {
		got := extractPath(tt.input)
		if got != tt.want {
			t.Errorf("extractPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPathsMatch(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		// Same path, different domains
		{"newgrounds.com/portal/view/59593", "flashpoint.example.com/portal/view/59593", true},
		// Different paths
		{"newgrounds.com/portal/view/59593", "newgrounds.com/portal/view/99999", false},
		// No path
		{"example.com", "example.com", false},
		// One has path, one doesn't
		{"example.com/foo", "example.com", false},
	}

	for _, tt := range tests {
		got := pathsMatch(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("pathsMatch(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestBestMatch(t *testing.T) {
	fp := NewFlashpointClient()

	entries := []flashpointEntry{
		{ID: "1", Title: "Alien Hominid", Developer: "Tom Fulp", Source: "https://www.newgrounds.com/portal/view/59593", Platform: "Flash"},
		{ID: "2", Title: "Alien Hominid HD", Developer: "Other", Source: "https://example.com/other", Platform: "Flash"},
		{ID: "3", Title: "Alien Thing", Developer: "Someone", Source: "https://example.com/alien", Platform: "Flash"},
	}

	// Exact URL match should win
	match := fp.bestMatch(entries, "alien hominid", "https://www.newgrounds.com/portal/view/59593")
	if match == nil {
		t.Fatal("expected match, got nil")
	}
	if match.ID != "1" {
		t.Errorf("expected ID=1 (URL match), got ID=%s", match.ID)
	}

	// Title exact match when URL doesn't match
	match = fp.bestMatch(entries, "Alien Hominid", "https://example.com/unknown")
	if match == nil {
		t.Fatal("expected match, got nil")
	}
	if match.ID != "1" {
		t.Errorf("expected ID=1 (title match), got ID=%s", match.ID)
	}

	// Title substring match
	match = fp.bestMatch(entries, "Alien Hominid HD", "")
	if match == nil {
		t.Fatal("expected match, got nil")
	}
	if match.ID != "2" {
		t.Errorf("expected ID=2 (exact title), got ID=%s", match.ID)
	}

	// No match at all
	match = fp.bestMatch(entries, "Totally Different Game", "https://example.com/nope")
	if match != nil {
		t.Errorf("expected nil, got match ID=%s", match.ID)
	}

	// Empty source URL, should fall back to title matching
	match = fp.bestMatch(entries, "Alien Hominid", "")
	if match == nil {
		t.Fatal("expected match, got nil")
	}
	if match.ID != "1" {
		t.Errorf("expected ID=1, got ID=%s", match.ID)
	}
}
