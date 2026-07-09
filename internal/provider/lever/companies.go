package lever

import (
	_ "embed"
	"fmt"
	"sort"

	"github.com/goccy/go-yaml"
)

//go:embed companies.yaml
var companiesYAML []byte

// Company is a verified Lever site from the curated roster. Site is the
// lowercase slug that namespaces its postings.
type Company struct {
	Name string `yaml:"company" json:"company"`
	Site string `yaml:"site" json:"site"`
}

// Companies holds every confirmed Lever tenant, sorted by company name.
var Companies = mustLoadCompanies()

// CompaniesBySite indexes companies by their lowercase site slug.
var CompaniesBySite = buildSiteIndex(Companies)

// mustLoadCompanies parses the package-owned embedded roster.
func mustLoadCompanies() []Company {
	var cs []Company
	if err := yaml.Unmarshal(companiesYAML, &cs); err != nil {
		panic(fmt.Sprintf("lever: parse companies.yaml: %v", err))
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].Name < cs[j].Name })
	return cs
}

func buildSiteIndex(cs []Company) map[string]Company {
	m := make(map[string]Company, len(cs))
	for _, c := range cs {
		m[c.Site] = c
	}
	return m
}
