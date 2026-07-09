package ashby

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

//go:embed companies.yaml
var companiesYAML []byte

// Company is a verified Ashby board from the curated roster. Board is the
// slug used as the API's jobBoardName.
type Company struct {
	Name  string `yaml:"company" json:"company"`
	Board string `yaml:"board" json:"board"`
}

// BoardURL returns the human-facing job board URL.
func (c Company) BoardURL() string {
	return fmt.Sprintf("https://jobs.ashbyhq.com/%s", c.Board)
}

// Companies holds every confirmed Ashby board, sorted by company name.
var Companies = mustLoadCompanies()

// CompaniesByBoard indexes companies by lowercase board slug.
var CompaniesByBoard = buildBoardIndex(Companies)

// mustLoadCompanies parses the package-owned embedded roster.
func mustLoadCompanies() []Company {
	var cs []Company
	if err := yaml.Unmarshal(companiesYAML, &cs); err != nil {
		panic(fmt.Sprintf("ashby: parse companies.yaml: %v", err))
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].Name < cs[j].Name })
	return cs
}

func buildBoardIndex(cs []Company) map[string]Company {
	m := make(map[string]Company, len(cs))
	for _, c := range cs {
		m[strings.ToLower(c.Board)] = c
	}
	return m
}
