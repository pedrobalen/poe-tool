package importers

import (
	"net/url"
	"strings"
)

// hostMatches reports whether host equals domain or is a subdomain of it,
// ignoring a leading "www." and any port.
func hostMatches(host, domain string) bool {
	host = strings.ToLower(host)
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	host = strings.TrimPrefix(host, "www.")

	return host == domain || strings.HasSuffix(host, "."+domain)
}

// firstPathSegment returns the first non-empty segment of a URL path.
func firstPathSegment(path string) string {
	for _, seg := range strings.Split(path, "/") {
		if seg != "" {
			return seg
		}
	}

	return ""
}

// looksLikeURL reports whether input parses as an absolute http(s) URL.
func looksLikeURL(input string) bool {
	u, err := url.Parse(strings.TrimSpace(input))
	if err != nil {
		return false
	}

	return u.Scheme == "http" || u.Scheme == "https"
}
