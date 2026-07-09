package job104

import (
	"net/url"
	"strings"
)

// JobCodeFromURL extracts the public job code. The search response's jobNo is
// an internal ID and cannot be passed to GetJobDetail.
func JobCodeFromURL(raw string) string {
	path := raw
	if u, err := url.Parse(raw); err == nil {
		path = u.Path
	}
	path = strings.TrimRight(path, "/")
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
