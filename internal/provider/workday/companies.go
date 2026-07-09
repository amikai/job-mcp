package workday

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

//go:embed companies.yaml
var companiesYAML []byte

// Company is a verified Workday CXS tenant from the curated roster. Tenant is
// the lowercase slug used to build its API URL.
type Company struct {
	Name     string `yaml:"company" json:"company"`
	Tenant   string `yaml:"tenant" json:"tenant"`
	Instance string `yaml:"instance" json:"instance"`
	Site     string `yaml:"site" json:"site"`
}

// BaseURL builds the company's Workday CXS base URL.
func (c Company) BaseURL() string {
	return fmt.Sprintf("https://%s.%s.myworkdayjobs.com/wday/cxs/%s/%s", c.Tenant, c.Instance, c.Tenant, c.Site)
}

// Companies holds every confirmed Workday tenant, sorted by company name.
var Companies = mustLoadCompanies()

// CompaniesByTenant indexes companies by lowercase tenant slug.
var CompaniesByTenant = buildTenantIndex(Companies)

// mustLoadCompanies parses the package-owned embedded roster.
func mustLoadCompanies() []Company {
	var cs []Company
	if err := yaml.Unmarshal(companiesYAML, &cs); err != nil {
		panic(fmt.Sprintf("workday: parse companies.yaml: %v", err))
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].Name < cs[j].Name })
	return cs
}

func buildTenantIndex(cs []Company) map[string]Company {
	m := make(map[string]Company, len(cs))
	for _, c := range cs {
		m[strings.ToLower(c.Tenant)] = c
	}
	return m
}
