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

// Filename takes a raw string and returns a safe filename. It strips path
// separators, collapses whitespace, removes control characters, and truncates
// to a reasonable length. The extension is preserved if provided separately.
func Filename(name, ext string) string {
	// Strip any directory components someone might try to sneak in.
	name = filepath.Base(name)

	// Decode common HTML entities that show up in scraped titles.
	name = decodeEntities(name)

	// Drop anything that could cause problems on Windows or Unix.
	name = unsafeChars.ReplaceAllString(name, "_")

	// Normalize whitespace to single underscores.
	name = multiSpace.ReplaceAllString(name, "_")
	name = multiUnderscore.ReplaceAllString(name, "_")

	// Trim leading/trailing junk.
	name = strings.Trim(name, "._- ")

	// Guard against empty or reserved names.
	if name == "" || name == "." || name == ".." {
		name = "untitled"
	}

	// Make sure the extension starts with a dot.
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	// Truncate to keep the full path under filesystem limits. 200 leaves
	// plenty of room for the directory portion and the extension.
	const maxBase = 200
	if len(name) > maxBase {
		name = name[:maxBase]
		name = strings.TrimRight(name, "_- ")
	}

	return name + ext
}

// URL checks that a raw URL string is a valid HTTP or HTTPS URL and returns
// the parsed form. Anything else is rejected outright.
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

// decodeEntities handles the handful of HTML entities that commonly appear in
// scraped page titles. A full HTML parser would be overkill here.
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

// StripNonPrintable removes non-printable unicode characters while keeping
// standard ASCII and common international characters intact.
func StripNonPrintable(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, s)
}
