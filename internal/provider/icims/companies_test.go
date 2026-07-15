package icims

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
		assert.True(t, strings.HasSuffix(c.Host, ".icims.com"))
		assert.Equal(t, strings.ToLower(c.Host), c.Host)
		_, ok := CompaniesByHost[c.Host]
		assert.True(t, ok, "host index missing %q", c.Host)
	}
	// Sorted by name.
	for i := 1; i < len(Companies); i++ {
		assert.LessOrEqual(t, Companies[i-1].Name, Companies[i].Name)
	}
}

func TestLoadCompaniesRejectsBadHost(t *testing.T) {
	_, err := loadCompanies([]byte(`- company: "X"
  host: "not-icims.example.com"
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), ".icims.com")
}

func TestCareersURL(t *testing.T) {
	c := Company{Name: "Peraton", Host: "careers-peraton.icims.com"}
	assert.Equal(t, "https://careers-peraton.icims.com/jobs/search?ss=1", c.CareersURL())
}
