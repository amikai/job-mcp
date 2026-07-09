package nvidia

import "strings"

// SplitExternalPath splits ExternalPath into the two path segments required by
// GetJobDetail; a combined parameter would URI-escape the separator.
func SplitExternalPath(externalPath string) (location, titleSlug string, ok bool) {
	location, titleSlug, ok = strings.Cut(strings.TrimPrefix(externalPath, "/job/"), "/")
	return location, titleSlug, ok
}
