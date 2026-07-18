package bamboohr

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompaniesLoaded(t *testing.T) {
	require.NotEmpty(t, Companies)
	assert.True(t, slicesAreSortedByName(Companies), "companies must be sorted by name")
	for _, c := range Companies {
		assert.Equal(t, c, CompaniesBySlug[c.Slug])
		assert.Equal(t, "https://"+c.Slug+".bamboohr.com/careers", c.CareersURL())
	}
}

func slicesAreSortedByName(cs []Company) bool {
	for i := 1; i < len(cs); i++ {
		if strings.Compare(cs[i-1].Name, cs[i].Name) > 0 {
			return false
		}
	}
	return true
}

func TestLoadCompaniesValidation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "duplicate slug",
			yaml:    "- company: A\n  slug: a\n- company: B\n  slug: a\n",
			wantErr: "duplicate slug",
		},
		{
			name:    "duplicate name",
			yaml:    "- company: A\n  slug: a\n- company: a\n  slug: b\n",
			wantErr: "duplicate company name",
		},
		{
			name:    "missing name",
			yaml:    "- company: \"\"\n  slug: a\n",
			wantErr: "company name is required",
		},
		{
			name:    "missing slug",
			yaml:    "- company: A\n  slug: \"\"\n",
			wantErr: "slug is required",
		},
		{
			name:    "uppercase slug",
			yaml:    "- company: A\n  slug: Acme\n",
			wantErr: "must be lowercase",
		},
		{
			name:    "invalid slug",
			yaml:    "- company: A\n  slug: \"a/b\"\n",
			wantErr: "invalid slug",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadCompanies([]byte(tt.yaml))
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
