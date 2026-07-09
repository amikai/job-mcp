package job104

import "net/http"

// BrowserTransport adds the headers 104's Cloudflare edge requires for JSON.
type BrowserTransport struct {
	Base http.RoundTripper
}

func (t BrowserTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	// Cloudflare requires a same-origin Referer; the site root works for all paths.
	req.Header.Set("Referer", "https://www.104.com.tw/")
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}
