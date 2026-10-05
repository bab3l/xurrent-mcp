package livefind

import (
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	defaultListPageLimit = 50
	hardMaxListPages     = 500
	envMaxPages          = "XURRENT_LIVEFIND_MAX_PAGES"
)

// linkNextPair matches one RFC 5988 link-param block pointing to rel=next.
var linkNextPair = regexp.MustCompile(`<([^>\s]+)>\s*;\s*rel\s*=\s*"?next"?`)

// listPageLimit returns max cursor pages per list (env XURRENT_LIVEFIND_MAX_PAGES, default 50, cap 500).
func listPageLimit() int {
	s := strings.TrimSpace(os.Getenv(envMaxPages))
	if s == "" {
		return defaultListPageLimit
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return defaultListPageLimit
	}
	if n > hardMaxListPages {
		return hardMaxListPages
	}
	return n
}

// linkHeaderBlocks returns all Link header field-values (HTTP allows multiple Link lines).
func linkHeaderBlocks(resp *http.Response) []string {
	if resp == nil {
		return nil
	}
	return resp.Header.Values("Link")
}

// nextSearchAfter extracts the search_after cursor from Link: rel="next" (or rel=next).
func nextSearchAfter(resp *http.Response) string {
	for _, block := range linkHeaderBlocks(resp) {
		for _, m := range linkNextPair.FindAllStringSubmatch(block, -1) {
			raw := strings.TrimSpace(m[1])
			if raw == "" {
				continue
			}
			// Relative paths from the API (e.g. /v1/teams?...) need a base for url.Parse query.
			if strings.HasPrefix(raw, "/") {
				raw = "https://api.xurrent.com" + raw
			}
			u, err := url.Parse(raw)
			if err != nil {
				continue
			}
			if q := u.Query().Get("search_after"); q != "" {
				return q
			}
		}
	}
	return ""
}
