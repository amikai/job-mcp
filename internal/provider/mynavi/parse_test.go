package mynavi

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func docFromString(t *testing.T, s string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(s))
	require.NoError(t, err)
	return doc
}

// A page without the result counter is not the search-results template —
// most notably the unfiltered legacy fallback page the site serves when the
// URL prefix is malformed — and must error instead of reading as zero hits.
func TestParseJobsHTMLUnknownPageErrors(t *testing.T) {
	doc := docFromString(t, `<html><body><p>検索結果一覧</p><div>55549件</div></body></html>`)
	_, err := parseJobsHTML(doc)
	assert.ErrorContains(t, err, "no result counter")
}

func TestParseJobsHTMLBadCounterErrors(t *testing.T) {
	doc := docFromString(t, `<html><body><p class="result__num"><em>N/A</em></p></body></html>`)
	_, err := parseJobsHTML(doc)
	assert.ErrorContains(t, err, "not a number")
}

// jobLocation arrives as an array on the fixture posting, but schema.org
// permits a bare object for a single location; both shapes must parse.
func TestParseJobDetailHTMLSingleLocation(t *testing.T) {
	doc := docFromString(t, `<html><head><script type="application/ld+json">
		{"@type":"JobPosting","title":"t",
		 "jobLocation":{"@type":"Place","address":{"@type":"PostalAddress","addressRegion":"東京都"}},
		 "baseSalary":{"@type":"MonetaryAmount","currency":"JPY","value":{"@type":"QuantitativeValue","minValue":4000000,"maxValue":8000000,"unitText":"YEAR"}}}
	</script></head></html>`)
	got, err := parseJobDetailHTML(doc, "1-1-1-1")
	require.NoError(t, err)
	assert.Equal(t, []Location{{Region: "東京都"}}, got.Locations)
	// bare-number salary values must parse too
	assert.Equal(t, "4000000", got.SalaryMin)
	assert.Equal(t, "8000000", got.SalaryMax)
}

// The detail page carries a BreadcrumbList JSON-LD before/after the
// JobPosting; selection must go by @type, not position.
func TestParseJobDetailHTMLSkipsOtherJSONLD(t *testing.T) {
	doc := docFromString(t, `<html><head>
		<script type="application/ld+json">{"@type":"BreadcrumbList"}</script>
		<script type="application/ld+json">{"@type":"JobPosting","title":"エンジニア"}</script>
	</head></html>`)
	got, err := parseJobDetailHTML(doc, "1-1-1-1")
	require.NoError(t, err)
	assert.Equal(t, "エンジニア", got.Title)
}

func TestParseJobDetailHTMLNoJSONLD(t *testing.T) {
	doc := docFromString(t, `<html><body><h1>404</h1></body></html>`)
	_, err := parseJobDetailHTML(doc, "1-1-1-1")
	assert.ErrorContains(t, err, "no JobPosting JSON-LD")
}

func TestHTMLToText(t *testing.T) {
	assert.Equal(t, "a\nb", htmlToText("a<br>b"))
	assert.Equal(t, "a\n\nb", htmlToText("<p>a</p><p>b</p>"))
	assert.Equal(t, "改行\nあり", htmlToText("改行&amp;lt;br&amp;gt;あり"), "double-escaped upstream br")
	assert.Equal(t, "plain", htmlToText("plain"))
	assert.Equal(t, "", htmlToText(""))
}
