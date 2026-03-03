package sanitize

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var (
	unsafeChars  = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	multiSpace   = regexp.MustCompile(`\s+`)
	multiUnderscore = regexp.MustCompile(`_{2,}`)
)

// Filename turns a raw string into something safe to use as a filename across
// platforms. The extension is appended separately so it doesn't get mangled.
func Filename(name, ext string) string {
	name = filepath.Base(name)
	name = decodeEntities(name)
	name = unsafeChars.ReplaceAllString(name, "_")
	name = multiSpace.ReplaceAllString(name, "_")
	name = multiUnderscore.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._- ")

	if name == "" || name == "." || name == ".." {
		name = "untitled"
	}

	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	const maxBase = 200
	if len(name) > maxBase {
		name = name[:maxBase]
		name = strings.TrimRight(name, "_- ")
	}

	return name + ext
}

// URL validates that a string is an HTTP(S) URL with a host.
func URL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)

	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http", "https":
		// Fine.
	default:
		return nil, &url.Error{Op: "parse", URL: raw, Err: errBadScheme}
	}

	if u.Host == "" {
		return nil, &url.Error{Op: "parse", URL: raw, Err: errNoHost}
	}

	return u, nil
}

type constError string

func (e constError) Error() string { return string(e) }

const (
	errBadScheme constError = "only http and https URLs are supported"
	errNoHost    constError = "URL is missing a host"
)

// decodeEntities handles common HTML entities from scraped page titles.
func decodeEntities(s string) string {
	r := strings.NewReplacer(
		"&#039;", "'",
		"&#39;", "'",
		"&apos;", "'",
		"&amp;", "and",
		"&quot;", "",
		"&lt;", "",
		"&gt;", "",
	)
	return r.Replace(s)
}

// StripNonPrintable drops non-printable unicode characters.
func StripNonPrintable(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, s)
}
