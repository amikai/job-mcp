package job104

import "testing"

func TestJobCodeFromURL(t *testing.T) {
	got := JobCodeFromURL("https://www.104.com.tw/job/abc123?jobsource=foo")
	if got != "abc123" {
		t.Fatalf("JobCodeFromURL() = %q", got)
	}
}
