package teamtailor

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type YCCompany struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type rosterCompany struct {
	Company string `yaml:"company"`
	Host    string `yaml:"host"`
}

func TestDiscoverNewCompanies(t *testing.T) {
	if os.Getenv("RUN_DISCOVERY") == "" {
		t.Skip("Skipping discovery test. Set RUN_DISCOVERY=1 to run.")
	}

	companyNames := make(map[string]bool)

	// 1. Load from other rosters
	t.Log("Loading company names from other rosters...")
	rosterFiles, err := filepath.Glob("../*/companies.yaml")
	if err != nil {
		t.Fatalf("Failed to search roster files: %v", err)
	}

	for _, file := range rosterFiles {
		// Skip teamtailor's own companies.yaml
		if strings.Contains(file, "teamtailor/companies.yaml") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Logf("Warning: failed to read %s: %v", file, err)
			continue
		}
		var comps []rosterCompany
		if err := yaml.Unmarshal(data, &comps); err != nil {
			t.Logf("Warning: failed to unmarshal %s: %v", file, err)
			continue
		}
		for _, c := range comps {
			if c.Company != "" {
				companyNames[c.Company] = true
			}
		}
	}
	t.Logf("Loaded %d company names from other rosters.", len(companyNames))

	// 2. Fetch connor11528 CSV
	t.Log("Fetching connor11528/tech-companies-and-startups CSV...")
	csvURL := "https://raw.githubusercontent.com/connor11528/tech-companies-and-startups/master/companies.csv"
	resp, err := http.Get(csvURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		reader := csv.NewReader(resp.Body)
		// Read header
		if _, err := reader.Read(); err == nil {
			for {
				row, err := reader.Read()
				if err != nil {
					break
				}
				if len(row) > 1 && row[1] != "" {
					companyNames[row[1]] = true
				}
			}
		}
		resp.Body.Close()
	} else {
		t.Logf("Warning: failed to fetch tech-companies CSV: %v", err)
	}
	t.Logf("Total company names after CSV: %d", len(companyNames))

	// 3. Fetch YC OSS API JSON
	t.Log("Fetching YC OSS API JSON...")
	ycURL := "https://yc-oss.github.io/api/companies/all.json"
	resp, err = http.Get(ycURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		var ycComps []YCCompany
		if err := json.NewDecoder(resp.Body).Decode(&ycComps); err == nil {
			for _, yc := range ycComps {
				if yc.Name != "" {
					companyNames[yc.Name] = true
				}
				if yc.Slug != "" {
					companyNames[yc.Slug] = true
				}
			}
		} else {
			t.Logf("Warning: failed to decode YC JSON: %v", err)
		}
		resp.Body.Close()
	} else {
		t.Logf("Warning: failed to fetch YC JSON: %v", err)
	}
	t.Logf("Total company names after YC JSON: %d", len(companyNames))

	// Generate candidate slugs
	candidateSlugs := make(map[string]bool)
	cleanReg := regexp.MustCompile("[^a-z0-9]")
	hyphenReg := regexp.MustCompile("[^a-z0-9]+")

	for name := range companyNames {
		nameLower := strings.TrimSpace(strings.ToLower(name))
		if nameLower == "" {
			continue
		}

		// Variation 1: Clean all non-alphanumeric
		s1 := cleanReg.ReplaceAllString(nameLower, "")
		if s1 != "" {
			candidateSlugs[s1] = true
		}

		// Variation 2: Replace non-alphanumeric with hyphens
		s2 := strings.Trim(hyphenReg.ReplaceAllString(nameLower, "-"), "-")
		if s2 != "" {
			candidateSlugs[s2] = true
		}

		// Variation 3: First word if multi-word
		words := strings.Fields(nameLower)
		if len(words) > 1 {
			s3 := cleanReg.ReplaceAllString(words[0], "")
			if len(s3) > 2 {
				candidateSlugs[s3] = true
			}
		}
	}

	// Generate candidate hosts and filter against current roster
	candidateHosts := make(map[string]bool)
	for slug := range candidateSlugs {
		host1 := slug + ".teamtailor.com"
		host2 := slug + ".na.teamtailor.com"

		if _, ok := CompaniesByHost[host1]; !ok {
			candidateHosts[host1] = true
		}
		if _, ok := CompaniesByHost[host2]; !ok {
			candidateHosts[host2] = true
		}
	}
	t.Logf("Generated %d candidate hosts to probe.", len(candidateHosts))

	// Channel to distribute hosts
	hostsChan := make(chan string, 100)
	var wg sync.WaitGroup

	// Client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Output streaming lock
	var outputMu sync.Mutex

	// Start workers
	numWorkers := 80
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for host := range hostsChan {
				url := fmt.Sprintf("https://%s/jobs.json", host)
				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					continue
				}
				req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

				resp, err := client.Do(req)
				if err != nil {
					continue
				}
				if resp.StatusCode == http.StatusOK {
					var feed struct {
						Title string `json:"title"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&feed); err == nil {
						title := strings.TrimSpace(feed.Title)
						if title != "" {
							outputMu.Lock()
							// Print immediately to stdout for streaming
							fmt.Printf("[FOUND] %s (%s)\n", title, host)
							outputMu.Unlock()
						}
					}
				}
				resp.Body.Close()
			}
		}()
	}

	// Feed hosts
	for host := range candidateHosts {
		hostsChan <- host
	}
	close(hostsChan)

	// Wait for workers
	wg.Wait()
}
