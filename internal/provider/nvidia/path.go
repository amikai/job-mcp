package nvidia

import "strings"

// SplitExternalPath splits a JobSummary.ExternalPath (e.g.
// "/job/US-CA-Remote/Software-Engineer--CUDA-Q_JR2011649") into the two path
// segments GetJobDetail expects. The API rejects a single combined path
// parameter because standard URI encoders escape the "/" between them.
func SplitExternalPath(externalPath string) (location, titleSlug string, ok bool) {
	location, titleSlug, ok = strings.Cut(strings.TrimPrefix(externalPath, "/job/"), "/")
	return location, titleSlug, ok
}
