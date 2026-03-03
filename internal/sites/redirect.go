package sites

import (
	"fmt"
	"net/http"
)

// safeRedirectPolicy blocks redirects to non-HTTP(S) schemes and caps the chain at 10.
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
