package ultipro

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractOpportunityDetailNotFound(t *testing.T) {
	// The genuine not-found shape: no marker anywhere in the page.
	_, err := extractOpportunityDetail([]byte(`<!DOCTYPE html><html><body>no posting here</body></html>`))
	require.ErrorIs(t, err, ErrJobNotFound)
}

func TestExtractOpportunityDetailValid(t *testing.T) {
	html := `<script>new US.Opportunity.CandidateOpportunityDetail({"Id":"abc-123","Title":"Engineer"});</script>`
	d, err := extractOpportunityDetail([]byte(html))
	require.NoError(t, err)
	assert.Equal(t, "abc-123", d.ID)
	assert.Equal(t, "Engineer", d.Title)
}

// TestExtractOpportunityDetailMalformedIsNotNotFound covers the marker
// being present but the payload broken in each way the parser can fail —
// none of these are the not-found case, so none may be ErrJobNotFound.
func TestExtractOpportunityDetailMalformedIsNotNotFound(t *testing.T) {
	cases := map[string]string{
		"unbalanced braces": `new US.Opportunity.CandidateOpportunityDetail({"Id":"abc-123","Title":"Engineer"`,
		"invalid JSON":      `new US.Opportunity.CandidateOpportunityDetail({"Id":"abc-123","Title":});`,
		"missing id/title":  `new US.Opportunity.CandidateOpportunityDetail({"RequisitionNumber":"R1"});`,
	}
	for name, html := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := extractOpportunityDetail([]byte(html))
			require.Error(t, err)
			assert.False(t, errors.Is(err, ErrJobNotFound), "malformed detail must not be reported as not-found")
		})
	}
}
