package workday

import (
	"fmt"
	"net/url"
	"strings"
)

// JobDetailKeyFromPath extracts the [GetJobDetailParams] values from
// [JobSummary.ExternalPath]. Two shapes are accepted:
//
//   - "/job/{location}/{titleSlug}" — the common form (e.g.
//     "/job/US-CA-Remote/Software-Engineer--CUDA_JR12345")
//   - "/job/{titleSlug}" — location-less paths some tenants emit (e.g.
//     "/job/APSCA-Certified-Social-Compliance-Auditor_JR0019413"). The
//     empty location is encoded as a zero-length path segment
//     (`/job//{titleSlug}`), which the CXS API accepts.
//
// A single combined path parameter fails because standard URI encoders
// escape the "/" between segments, so callers need the parts split
// apart. Anything else — a missing "/job/" prefix, an empty titleSlug, or
// extra path segments — returns ok=false so callers can fall back
// instead of sending a request that's guaranteed to fail.
func JobDetailKeyFromPath(externalPath string) (location, titleSlug string, ok bool) {
	rest, found := strings.CutPrefix(externalPath, "/job/")
	if !found || rest == "" {
		return "", "", false
	}
	location, titleSlug, cut := strings.Cut(rest, "/")
	if !cut {
		// Location-less: "/job/{titleSlug}".
		return "", location, true
	}
	if titleSlug == "" || strings.Contains(titleSlug, "/") {
		return "", "", false
	}
	// "/job//{titleSlug}" (empty location segment) is accepted; a
	// non-empty location is the typical two-segment form.
	return location, titleSlug, true
}

// PublicSiteURL derives a Workday tenant's public-facing (non-API) career
// site origin from its CXS base URL. It takes the base URL's last path
// segment, the "{site}" segment shared by both URL shapes (see
// openapi.yaml's "Multi-tenant URL shape" note). For example:
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
