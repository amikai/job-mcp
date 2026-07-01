package job104

import "net/http"

// BrowserTransport adds the headers 104's edge (Cloudflare) requires to see
// on every request, or it returns an HTML challenge page instead of JSON.
// Wrap it around an *http.Client and pass that to NewClient via WithClient.
type BrowserTransport struct {
	Base http.RoundTripper
}

func (t BrowserTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	// A same-origin Referer is required — Cloudflare 403s otherwise; a
	// generic one (not the exact source page) is accepted for every path.
	req.Header.Set("Referer", "https://www.104.com.tw/")
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}
