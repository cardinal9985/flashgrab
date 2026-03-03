package sites

import (
	"strings"
	"testing"
)

func TestStripJSFunctions(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string // after stripping, should be valid JSON
	}{
		{
			"no functions",
			`[{"url":"https://example.com/game.swf","description":"Flash"}]`,
			`[{"url":"https://example.com/game.swf","description":"Flash"}]`,
		},
		{
			"callback function",
			`[{"url":"https://example.com/game.zip","description":"HTML5",callback:function(){ var x = 1; }}]`,
			`[{"url":"https://example.com/game.zip","description":"HTML5"}]`,
		},
		{
			"multiple unquoted keys",
			`[{"url":"https://example.com/game.zip",callback:function(){},html:"<div></div>"}]`,
			`[{"url":"https://example.com/game.zip"}]`,
		},
		{
			"nested braces in function",
			`[{"url":"https://example.com/game.zip",callback:function(){ if(true){ var x = {a:1}; } }}]`,
			`[{"url":"https://example.com/game.zip"}]`,
		},
		{
			"empty params object",
			`[{"url":"test","params":{},callback:function(){}}]`,
			`[{"url":"test","params":{}}]`,
		},
		{
			"empty array field",
			`[{"url":"test","items":[],callback:function(){}}]`,
			`[{"url":"test","items":[]}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripJSFunctions(tt.in)
			if got != tt.want {
				t.Errorf("stripJSFunctions:\n  got:  %s\n  want: %s", got, tt.want)
			}
		})
	}
}

func TestParseEmbeds(t *testing.T) {
	ng := &newgrounds{}

	t.Run("standard flash embed", func(t *testing.T) {
		page := `<html><script>var embed_controller = new embedController([{"url":"https://uploads.ungrounded.net/game.swf","description":"Flash Game","width":800,"height":600,"filesize":1234}], null);</script></html>`
		embeds, err := ng.parseEmbeds(page)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(embeds) != 1 {
			t.Fatalf("expected 1 embed, got %d", len(embeds))
		}
		if embeds[0].URL != "https://uploads.ungrounded.net/game.swf" {
			t.Errorf("unexpected URL: %s", embeds[0].URL)
		}
		if embeds[0].Filesize != 1234 {
			t.Errorf("unexpected filesize: %d", embeds[0].Filesize)
		}
	})

	t.Run("html5 embed with callback", func(t *testing.T) {
		page := `<script>var embed_controller = new embedController([{"url":"https://uploads.ungrounded.net/game.zip","is_published":true,"description":"HTML5 Archive","width":750,"height":600,"filesize":5000,"params":{},callback:function(){ var success = true; if(embed_controller.isCompatible()){ success = true; }}}], null, false);</script>`
		embeds, err := ng.parseEmbeds(page)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(embeds) != 1 {
			t.Fatalf("expected 1 embed, got %d", len(embeds))
		}
		if embeds[0].URL != "https://uploads.ungrounded.net/game.zip" {
			t.Errorf("unexpected URL: %s", embeds[0].URL)
		}
		if embeds[0].Description != "HTML5 Archive" {
			t.Errorf("unexpected description: %s", embeds[0].Description)
		}
	})

	t.Run("old format with trailing function", func(t *testing.T) {
		// Older pages might use embedController([...], function callback)
		page := `<script>embedController([{"url":"https://uploads.ungrounded.net/old.swf","description":"Flash","width":640,"height":480}], function onReady() {})</script>`
		embeds, err := ng.parseEmbeds(page)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(embeds) != 1 {
			t.Fatalf("expected 1 embed, got %d", len(embeds))
		}
		if embeds[0].URL != "https://uploads.ungrounded.net/old.swf" {
			t.Errorf("unexpected URL: %s", embeds[0].URL)
		}
	})

	t.Run("no embed data", func(t *testing.T) {
		page := `<html><body>No game here</body></html>`
		_, err := ng.parseEmbeds(page)
		if err == nil {
			t.Error("expected error for page with no embed data")
		}
	})

	t.Run("multiple embeds", func(t *testing.T) {
		page := `<script>new embedController([{"url":"https://example.com/a.swf","description":"Game"},{"url":"https://example.com/b.swf","description":"Extras"}])</script>`
		embeds, err := ng.parseEmbeds(page)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(embeds) != 2 {
			t.Fatalf("expected 2 embeds, got %d", len(embeds))
		}
	})
}

func TestGuessExtension(t *testing.T) {
	tests := []struct {
		desc string
		want string
	}{
		{"Flash Game", ".swf"},
		{"HTML5 Archive", ".zip"},
		{"flash", ".swf"},
		{"html5", ".zip"},
		{"Unknown", ".swf"},
		{"", ".swf"},
	}
	for _, tt := range tests {
		got := guessExtension(tt.desc)
		if got != tt.want {
			t.Errorf("guessExtension(%q) = %q, want %q", tt.desc, got, tt.want)
		}
	}
}

func TestNgEmbedStartPattern(t *testing.T) {
	variants := []string{
		`embedController([`,
		`new embedController([`,
		`var x = new embedController([`,
	}
	for _, s := range variants {
		if !strings.Contains(s, "embedController(") {
			t.Errorf("expected embedController( in %q", s)
		}
		loc := ngEmbedStart.FindStringIndex(s)
		if loc == nil {
			t.Errorf("ngEmbedStart didn't match %q", s)
		}
	}
}
