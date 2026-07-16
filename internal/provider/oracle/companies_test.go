package oracle

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompaniesLoaded(t *testing.T) {
	require.NotEmpty(t, Companies)
	for _, c := range Companies {
		assert.NotEmpty(t, c.Name)
		assert.True(t, strings.HasSuffix(c.Host, ".oraclecloud.com"))
		assert.Equal(t, strings.ToLower(c.Host), c.Host)
		assert.NotEmpty(t, c.Site)
		assert.NotEmpty(t, c.SiteNumber)
		_, ok := CompaniesByKey[companyKey(c.Host, c.SiteNumber)]
		assert.True(t, ok, "key index missing %q/%q", c.Host, c.SiteNumber)
	}
	for i := 1; i < len(Companies); i++ {
		assert.LessOrEqual(t, Companies[i-1].Name, Companies[i].Name)
	}
}

func TestLoadCompaniesRejectsBadHost(t *testing.T) {
	tests := []struct {
		name string
		host string
	}{
		{name: "unrelated domain", host: "example.com"},
		{name: "spoofed suffix", host: "foo.oraclecloud.com.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte(`- company: "X"
  host: "` + tt.host + `"
  site: "CX_1"
  site_number: "CX_1"
`)
			_, err := loadCompanies(data)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "oraclecloud.com")
		})
	}
}

func TestLoadCompaniesRejectsDuplicateKey(t *testing.T) {
	_, err := loadCompanies([]byte(`
- company: "A"
  host: "jpmc.fa.oraclecloud.com"
  site: "CX_1001"
  site_number: "CX_1001"
- company: "B"
  host: "jpmc.fa.oraclecloud.com"
  site: "other"
  site_number: "CX_1001"
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate host/site_number")
}

func TestCareersURL(t *testing.T) {
	c := Company{
		Name:       "Mayo Clinic",
		Host:       "fa-euwp-saasfaprod1.fa.ocs.oraclecloud.com",
		Site:       "Mayo-US",
		SiteNumber: "CX_1",
	}
	assert.Equal(t, "https://fa-euwp-saasfaprod1.fa.ocs.oraclecloud.com/hcmUI/CandidateExperience/en/sites/Mayo-US/jobs", c.CareersURL())
	assert.Equal(t, "https://fa-euwp-saasfaprod1.fa.ocs.oraclecloud.com", c.APIBaseURL())
}
