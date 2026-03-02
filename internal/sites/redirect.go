package sites

import (
	"fmt"
	"net/http"
)

// safeRedirectPolicy prevents the HTTP client from following redirects to
// non-HTTP schemes. This blocks file://, ftp://, and other protocol handlers
// that could be abused through a malicious redirect.
func safeRedirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("too many redirects")
	}

	switch req.URL.Scheme {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("refusing to follow redirect to %s://", req.URL.Scheme)
	}
}
