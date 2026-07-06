//go:build ignore

// verify_companies concurrently re-checks every board in ../companies.yaml
// against the live Ashby posting API. A board is BAD if the API doesn't
// return a job-board response at all (org left Ashby / renamed its board);
// it's WARN if the response is valid but the jobs array is empty (possibly
// abandoned). Run with: go run testdata/verify_companies.go [-fix]
//
// -fix rewrites companies.yaml in place, dropping every BAD board.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"time"
)

type company struct {
	Name string
	Board string
}

type result struct {
	company company
	status  string // "OK", "WARN", "BAD"
	detail  string
}

var boardLineRe = regexp.MustCompile(`^- company: "([^"]+)"$`)
var boardValueRe = regexp.MustCompile(`^\s*board: "([^"]+)"$`)

func loadCompanies(path string) ([]company, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cs []company
	var pending string
	for _, line := range splitLines(string(data)) {
		if m := boardLineRe.FindStringSubmatch(line); m != nil {
			pending = m[1]
			continue
		}
		if m := boardValueRe.FindStringSubmatch(line); m != nil && pending != "" {
			cs = append(cs, company{Name: pending, Board: m[1]})
			pending = ""
		}
	}
	return cs, nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func verify(c company) result {
	client := &http.Client{Timeout: 30 * time.Second}
	url := "https://api.ashbyhq.com/posting-api/job-board/" + c.Board
	resp, err := client.Get(url)
	if err != nil {
		return result{c, "BAD", err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return result{c, "BAD", fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}
	var parsed struct {
		Jobs []json.RawMessage `json:"jobs"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || parsed.Jobs == nil {
		return result{c, "BAD", "not a job-board response"}
	}
	if len(parsed.Jobs) == 0 {
		return result{c, "WARN", "0 jobs (possibly abandoned board)"}
	}
	return result{c, "OK", fmt.Sprintf("%d jobs", len(parsed.Jobs))}
}

func writeCompanies(path string, cs []company) error {
	sort.Slice(cs, func(i, j int) bool { return cs[i].Name < cs[j].Name })
	var out []byte
	for _, c := range cs {
		out = append(out, fmt.Sprintf("- company: %q\n  board: %q\n", c.Name, c.Board)...)
	}
	return os.WriteFile(path, out, 0o644)
}

func main() {
	fix := flag.Bool("fix", false, "rewrite companies.yaml, dropping BAD boards")
	concurrency := flag.Int("c", 10, "max concurrent requests")
	flag.Parse()

	const path = "companies.yaml"
	companies, err := loadCompanies(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load companies.yaml:", err)
		os.Exit(1)
	}

	jobs := make(chan company)
	results := make(chan result)
	go func() {
		for _, c := range companies {
			jobs <- c
		}
		close(jobs)
	}()
	for i := 0; i < *concurrency; i++ {
		go func() {
			for c := range jobs {
				results <- verify(c)
			}
		}()
	}

	all := make([]result, 0, len(companies))
	for range companies {
		all = append(all, <-results)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].company.Name < all[j].company.Name })

	var bad, warn int
	for _, r := range all {
		fmt.Printf("%-4s %-24s %-16s %s\n", r.status, r.company.Name, r.company.Board, r.detail)
		switch r.status {
		case "BAD":
			bad++
		case "WARN":
			warn++
		}
	}
	fmt.Printf("\n%d ok, %d warn, %d bad (of %d)\n", len(all)-bad-warn, warn, bad, len(all))

	if *fix && bad > 0 {
		keep := make([]company, 0, len(companies))
		badSet := make(map[string]bool)
		for _, r := range all {
			if r.status == "BAD" {
				badSet[r.company.Board] = true
			}
		}
		for _, c := range companies {
			if !badSet[c.Board] {
				keep = append(keep, c)
			}
		}
		if err := writeCompanies(path, keep); err != nil {
			fmt.Fprintln(os.Stderr, "write companies.yaml:", err)
			os.Exit(1)
		}
		fmt.Printf("removed %d bad boards, wrote %d entries to %s\n", bad, len(keep), path)
	}

	if bad > 0 && !*fix {
		os.Exit(1)
	}
}
