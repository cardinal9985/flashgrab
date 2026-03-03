package sanitize

import (
	"testing"
)

func TestFilename(t *testing.T) {
	tests := []struct {
		name string
		in   string
		ext  string
		want string
	}{
		{"basic", "Cool Game", ".swf", "Cool_Game.swf"},
		{"html entities", "Tom&#039;s Adventure", ".swf", "Tom's_Adventure.swf"},
		{"ampersand", "Cops &amp; Robbers", ".zip", "Cops_and_Robbers.zip"},
		{"path traversal", "../../etc/passwd", ".swf", "passwd.swf"},
		{"windows reserved chars", `game<>:"name`, ".swf", "game_name.swf"},
		{"backslash", `game\name`, ".swf", "game_name.swf"}, // backslash replaced by unsafeChars on linux
		{"control chars", "game\x00\x01\x1fname", ".swf", "game_name.swf"},
		{"multiple spaces", "game    name", ".swf", "game_name.swf"},
		{"empty name", "", ".swf", "untitled.swf"},
		{"dot only", ".", ".swf", "untitled.swf"},
		{"dotdot", "..", ".swf", "untitled.swf"},
		{"ext without dot", "game", "swf", "game.swf"},
		{"leading junk", "...---game", ".swf", "game.swf"},
		{"trailing junk", "game...__", ".zip", "game.zip"},
		{"no ext", "Cool Game", "", "Cool_Game"},
		{"unicode preserved", "ゲーム名前", ".swf", "ゲーム名前.swf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Filename(tt.in, tt.ext)
			if got != tt.want {
				t.Errorf("Filename(%q, %q) = %q, want %q", tt.in, tt.ext, got, tt.want)
			}
		})
	}
}

func TestFilenameTruncation(t *testing.T) {
	long := ""
	for i := 0; i < 300; i++ {
		long += "a"
	}
	got := Filename(long, ".swf")
	if len(got) > 205 { // 200 base + ".swf"
		t.Errorf("expected truncation, got length %d", len(got))
	}
	if got[len(got)-4:] != ".swf" {
		t.Errorf("expected .swf extension, got %q", got[len(got)-4:])
	}
}

func TestURL(t *testing.T) {
	valid := []string{
		"https://www.newgrounds.com/portal/view/123",
		"http://example.com",
		"https://author.itch.io/game",
	}
	for _, raw := range valid {
		t.Run(raw, func(t *testing.T) {
			u, err := URL(raw)
			if err != nil {
				t.Fatalf("URL(%q) unexpected error: %v", raw, err)
			}
			if u.Host == "" {
				t.Errorf("URL(%q) returned empty host", raw)
			}
		})
	}

	invalid := []struct {
		name string
		raw  string
	}{
		{"ftp", "ftp://example.com/file.swf"},
		{"javascript", "javascript:alert(1)"},
		{"no scheme", "www.example.com"},
		{"empty", ""},
		{"file", "file:///etc/passwd"},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			_, err := URL(tt.raw)
			if err == nil {
				t.Errorf("URL(%q) expected error, got nil", tt.raw)
			}
		})
	}
}

func TestStripNonPrintable(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"hello world", "hello world"},
		{"hello\x00world", "helloworld"},
		{"tab\there", "tabhere"}, // tab is in \x00-\x1f range
	}
	for _, tt := range tests {
		got := StripNonPrintable(tt.in)
		if got != tt.want {
			t.Errorf("StripNonPrintable(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
