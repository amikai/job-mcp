package workday

import (
	"fmt"
	"net/url"
	"strings"
)

// JobDetailKeyFromPath splits a Workday ExternalPath into the two path
// parameters required by GetJobDetail. It rejects malformed or extra segments.
func JobDetailKeyFromPath(externalPath string) (location, titleSlug string, ok bool) {
	rest, found := strings.CutPrefix(externalPath, "/job/")
	if !found {
		return "", "", false
	}
	location, titleSlug, ok = strings.Cut(rest, "/")
	if !ok || location == "" || titleSlug == "" || strings.Contains(titleSlug, "/") {
		return "", "", false
	}
	return location, titleSlug, true
}

// PublicSiteURL derives the public career-site origin from a CXS base URL.
//
//	https://nvidia.wd5.myworkdayjobs.com/wday/cxs/nvidia/NVIDIAExternalCareerSite
//	  -> https://nvidia.wd5.myworkdayjobs.com/NVIDIAExternalCareerSite
func PublicSiteURL(baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base URL %q: %w", baseURL, err)
	}
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	site := segments[len(segments)-1]
	if site == "" {
		return "", fmt.Errorf("base URL %q has no path segment to derive a site from", baseURL)
	}
	return fmt.Sprintf("%s://%s/%s", u.Scheme, u.Host, site), nil
}
