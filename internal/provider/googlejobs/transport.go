package googlejobs

import "net/http"

const defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"

// BrowserTransport adds the browser headers used by the upstream JobSpy
// scraper. Its zero value delegates to http.DefaultTransport.
type BrowserTransport struct {
	Base http.RoundTripper
}

// RoundTrip adds defaults without overriding caller-provided headers.
func (t BrowserTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()

	setDefaultHeader(clone.Header, "Accept-Language", "en-US,en;q=0.9")
	setDefaultHeader(clone.Header, "Referer", "https://www.google.com/")
	setDefaultHeader(clone.Header, "Sec-CH-Prefers-Color-Scheme", "dark")
	setDefaultHeader(clone.Header, "Sec-CH-UA", `"Chromium";v="130", "Google Chrome";v="130", "Not?A_Brand";v="99"`)
	setDefaultHeader(clone.Header, "Sec-CH-UA-Arch", `"arm"`)
	setDefaultHeader(clone.Header, "Sec-CH-UA-Bitness", `"64"`)
	setDefaultHeader(clone.Header, "Sec-CH-UA-Form-Factors", `"Desktop"`)
	setDefaultHeader(clone.Header, "Sec-CH-UA-Full-Version", `"130.0.6723.58"`)
	setDefaultHeader(clone.Header, "Sec-CH-UA-Full-Version-List", `"Chromium";v="130.0.6723.58", "Google Chrome";v="130.0.6723.58", "Not?A_Brand";v="99.0.0.0"`)
	setDefaultHeader(clone.Header, "Sec-CH-UA-Mobile", "?0")
	setDefaultHeader(clone.Header, "Sec-CH-UA-Model", `""`)
	setDefaultHeader(clone.Header, "Sec-CH-UA-Platform", `"macOS"`)
	setDefaultHeader(clone.Header, "Sec-CH-UA-Platform-Version", `"15.0.1"`)
	setDefaultHeader(clone.Header, "Sec-CH-UA-WoW64", "?0")
	setDefaultHeader(clone.Header, "Sec-Fetch-Site", "same-origin")
	setDefaultHeader(clone.Header, "User-Agent", defaultUserAgent)

	if clone.URL.Path == "/async/callback:550" {
		setDefaultHeader(clone.Header, "Accept", "*/*")
		setDefaultHeader(clone.Header, "Priority", "u=1, i")
		setDefaultHeader(clone.Header, "Sec-Fetch-Dest", "empty")
		setDefaultHeader(clone.Header, "Sec-Fetch-Mode", "cors")
	} else {
		setDefaultHeader(clone.Header, "Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
		setDefaultHeader(clone.Header, "Priority", "u=0, i")
		setDefaultHeader(clone.Header, "Sec-Fetch-Dest", "document")
		setDefaultHeader(clone.Header, "Sec-Fetch-Mode", "navigate")
		setDefaultHeader(clone.Header, "Sec-Fetch-User", "?1")
		setDefaultHeader(clone.Header, "Upgrade-Insecure-Requests", "1")
		setDefaultHeader(clone.Header, "X-Browser-Channel", "stable")
		setDefaultHeader(clone.Header, "X-Browser-Copyright", "Copyright 2024 Google LLC. All rights reserved.")
		setDefaultHeader(clone.Header, "X-Browser-Year", "2024")
	}

	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(clone)
}

func setDefaultHeader(header http.Header, key, value string) {
	if header.Get(key) == "" {
		header.Set(key, value)
	}
}
