package greenhouse

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

//go:embed companies.yaml
var companiesYAML []byte

// Company is a verified Greenhouse board from the curated roster. BoardToken
// is the identifier used by the board_token API parameter.
type Company struct {
	Name       string `yaml:"company" json:"company"`
	BoardToken string `yaml:"board_token" json:"board_token"`
}

// BoardURL returns the human-facing job board URL.
func (c Company) BoardURL() string {
	return fmt.Sprintf("https://job-boards.greenhouse.io/%s", c.BoardToken)
}

// Companies holds every confirmed Greenhouse board, sorted by company name.
var Companies = mustLoadCompanies()

// CompaniesByBoardToken indexes companies by lowercase board token.
var CompaniesByBoardToken = buildBoardTokenIndex(Companies)

// mustLoadCompanies parses the package-owned embedded roster.
func mustLoadCompanies() []Company {
	var cs []Company
	if err := yaml.Unmarshal(companiesYAML, &cs); err != nil {
		panic(fmt.Sprintf("greenhouse: parse companies.yaml: %v", err))
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].Name < cs[j].Name })
	return cs
}

func buildBoardTokenIndex(cs []Company) map[string]Company {
	m := make(map[string]Company, len(cs))
	for _, c := range cs {
		m[strings.ToLower(c.BoardToken)] = c
	}
	return m
}
