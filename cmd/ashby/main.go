package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jaytaylor/html2text"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"

	ashby "github.com/amikai/openings-mcp/internal/provider/ashby"
)

// apiBaseURL is Ashby's public posting API origin.
const apiBaseURL = "https://api.ashbyhq.com"

func main() {
	rootFlags := ff.NewFlagSet("ashby")
	var (
		board   = rootFlags.StringLong("board", "", "confirmed Ashby board slug, e.g. openai (see 'ashby companies' for the full list)")
		timeout = rootFlags.DurationLong("timeout", 60*time.Second, "request timeout")
		format  = rootFlags.StringEnumLong("format", "output format", "text", "json")
	)
	rootCmd := &ff.Command{
		Name:  "ashby",
		Usage: "ashby --board BOARD [FLAGS] <companies|search|get> [FLAGS]",
		Flags: rootFlags,
	}

	companiesFlags := ff.NewFlagSet("companies").SetParent(rootFlags)
	companiesCmd := &ff.Command{
		Name:      "companies",
		Usage:     "ashby companies [--format text|json]",
		ShortHelp: "list confirmed Ashby boards (company name and board slug)",
		Flags:     companiesFlags,
		Exec: func(ctx context.Context, args []string) error {
			return runCompanies(*format)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, companiesCmd)

	searchFlags := ff.NewFlagSet("search").SetParent(rootFlags)
	keyword := searchFlags.StringLong("keyword", "", "case-insensitive substring filter on job titles (empty lists every job)")
	searchCmd := &ff.Command{
		Name:      "search",
		Usage:     "ashby --board BOARD search [--keyword TEXT] [--format text|json]",
		ShortHelp: "list a board's jobs as summaries (client-side keyword filter)",
		Flags:     searchFlags,
		Exec: func(ctx context.Context, args []string) error {
			return runSearch(ctx, *board, *timeout, *keyword, *format)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, searchCmd)

	getFlags := ff.NewFlagSet("get").SetParent(rootFlags)
	jobID := getFlags.StringLong("id", "", "job posting UUID from search results")
	getCmd := &ff.Command{
		Name:      "get",
		Usage:     "ashby --board BOARD get --id UUID [--format text|json]",
		ShortHelp: "print one job in full (description and compensation)",
		Flags:     getFlags,
		Exec: func(ctx context.Context, args []string) error {
			return runGet(ctx, *board, *timeout, *jobID, *format)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, getCmd)

	if err := rootCmd.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, ffhelp.Command(rootCmd.GetSelected()))
		if errors.Is(err, ff.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "err:", err)
		os.Exit(1)
	}

	if rootCmd.GetSelected() == rootCmd {
		fmt.Fprintln(os.Stderr, ffhelp.Command(rootCmd))
		fmt.Fprintln(os.Stderr, "err: a subcommand (companies, search, or get) is required")
		os.Exit(1)
	}

	if err := rootCmd.Run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "err:", err)
		os.Exit(1)
	}
}

// runCompanies prints the embedded Ashby board roster without a network call.
func runCompanies(format string) error {
	cs := ashby.Companies

	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(cs)
	}

	for _, c := range cs {
		fmt.Printf("%s (%s)\n", c.Name, c.Board)
	}
	return nil
}

// fetchBoard validates the board slug and fetches its full job list with
// compensation. A typed 404 is reported even though roster boards should not
// produce one.
func fetchBoard(ctx context.Context, board string, timeout time.Duration) (*ashby.JobBoardResponse, error) {
	if board == "" {
		return nil, fmt.Errorf("--board is required")
	}
	slug := strings.ToLower(board)
	if _, ok := ashby.CompaniesByBoard[slug]; !ok {
		return nil, fmt.Errorf("board %q not found; run 'ashby companies' to see supported boards", board)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := ashby.NewClient(apiBaseURL)
	if err != nil {
		return nil, err
	}

	res, err := client.GetJobBoard(ctx, ashby.GetJobBoardParams{
		JobBoardName:        slug,
		IncludeCompensation: ashby.NewOptBool(true),
	})
	if err != nil {
		return nil, err
	}
	switch r := res.(type) {
	case *ashby.JobBoardResponse:
		return r, nil
	case *ashby.GetJobBoardNotFound:
		return nil, fmt.Errorf("board %q not found upstream", board)
	default:
		return nil, fmt.Errorf("unexpected response type %T", res)
	}
}

// jobSummaryJSON is the compact --format json projection used by search and get.
type jobSummaryJSON struct {
	ID                 string   `json:"id,omitempty"`
	Title              string   `json:"title"`
	Department         string   `json:"department,omitempty"`
	Team               string   `json:"team,omitempty"`
	Location           string   `json:"location,omitempty"`
	SecondaryLocations []string `json:"secondaryLocations,omitempty"`
	WorkplaceType      string   `json:"workplaceType,omitempty"`
	IsRemote           *bool    `json:"isRemote,omitempty"`
	PublishedAt        string   `json:"publishedAt"`
	Compensation       string   `json:"compensation,omitempty"`
	URL                string   `json:"url"`
}

type searchResultJSON struct {
	Total int              `json:"total"`
	Jobs  []jobSummaryJSON `json:"jobs"`
}

func summarize(j ashby.JobPosting) jobSummaryJSON {
	s := jobSummaryJSON{
		ID:          j.ID.Value,
		Title:       j.Title,
		Department:  j.Department.Value,
		Team:        j.Team.Value,
		Location:    j.Location.Value,
		PublishedAt: j.PublishedAt.Format("2006-01-02"),
		URL:         j.JobUrl,
	}
	// Null fields stay omitted rather than defaulting to OnSite/false.
	if !j.WorkplaceType.Null {
		s.WorkplaceType = string(j.WorkplaceType.Value)
	}
	if !j.IsRemote.Null {
		v := j.IsRemote.Value
		s.IsRemote = &v
	}
	for _, sl := range j.SecondaryLocations {
		if sl.Location.Set {
			s.SecondaryLocations = append(s.SecondaryLocations, sl.Location.Value)
		}
	}
	if j.Compensation.Set {
		// A job can enable compensation without publishing any ranges.
		if v, ok := j.Compensation.Value.CompensationTierSummary.Get(); ok {
			s.Compensation = v
		}
	}
	return s
}

// runSearch fetches the whole board, filters titles locally, and prints the
// summaries. Ashby returns the complete board in one response.
func runSearch(ctx context.Context, board string, timeout time.Duration, keyword, format string) error {
	resp, err := fetchBoard(ctx, board, timeout)
	if err != nil {
		return err
	}

	matched := make([]jobSummaryJSON, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		if keyword != "" && !strings.Contains(strings.ToLower(j.Title), strings.ToLower(keyword)) {
			continue
		}
		matched = append(matched, summarize(j))
	}

	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(searchResultJSON{Total: len(resp.Jobs), Jobs: matched})
	}

	fmt.Printf("Ashby Jobs Report\n")
	fmt.Printf("Found %d jobs; showing %d\n\n", len(resp.Jobs), len(matched))
	for i, s := range matched {
		fmt.Printf("%d. %s\n", i+1, s.Title)
		printSummary(s)
		fmt.Println()
	}
	return nil
}

// printSummary renders the compact text block, using a list when a job has
// multiple locations.
func printSummary(s jobSummaryJSON) {
	switch {
	case s.Department != "" && s.Team != "" && s.Team != s.Department:
		fmt.Printf("Department: %s / %s\n", s.Department, s.Team)
	case s.Department != "":
		fmt.Printf("Department: %s\n", s.Department)
	}
	if len(s.SecondaryLocations) > 0 {
		fmt.Println("Locations:")
		if s.Location != "" {
			fmt.Printf("  - %s\n", s.Location)
		}
		for _, l := range s.SecondaryLocations {
			fmt.Printf("  - %s\n", l)
		}
	} else if s.Location != "" {
		fmt.Printf("Location: %s\n", s.Location)
	}
	// Some boards send both fields as null; do not invent a default.
	if s.WorkplaceType != "" || (s.IsRemote != nil && *s.IsRemote) {
		workplace := s.WorkplaceType
		if workplace == "" {
			workplace = "(unspecified)"
		}
		if s.IsRemote != nil && *s.IsRemote {
			workplace += " (remote)"
		}
		fmt.Printf("Workplace: %s\n", workplace)
	}
	fmt.Printf("Posted: %s\n", s.PublishedAt)
	if s.Compensation != "" {
		fmt.Printf("Compensation: %s\n", s.Compensation)
	}
	fmt.Printf("URL: %s\n", s.URL)
	if s.ID != "" {
		fmt.Printf("ID: %s\n", s.ID)
	}
}

// runGet fetches the whole board because Ashby has no per-job endpoint, then
// prints the matching job.
func runGet(ctx context.Context, board string, timeout time.Duration, jobID, format string) error {
	if jobID == "" {
		return fmt.Errorf("--id is required")
	}
	resp, err := fetchBoard(ctx, board, timeout)
	if err != nil {
		return err
	}
	for _, j := range resp.Jobs {
		if j.ID.Value == jobID {
			return printJob(j, format)
		}
	}
	return fmt.Errorf("job %q not found on board %q", jobID, board)
}

func printJob(j ashby.JobPosting, format string) error {
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(j)
	}

	s := summarize(j)
	fmt.Println(s.Title)
	printSummary(s)
	fmt.Printf("Employment: %s\n", j.EmploymentType)
	fmt.Printf("Apply: %s\n", j.ApplyUrl)
	if j.Compensation.Set {
		printCompensation(j.Compensation.Value)
	}

	desc := ""
	if v, ok := j.DescriptionHtml.Get(); ok {
		if text, err := html2text.FromString(v, html2text.Options{}); err == nil {
			desc = text
		}
	}
	if desc == "" {
		desc = j.DescriptionPlain.Value
	}
	if desc != "" {
		fmt.Printf("\nDescription:\n%s\n", desc)
	}
	return nil
}

// printCompensation renders the detail below the summary. Empty tier lists
// produce no output.
func printCompensation(c ashby.Compensation) {
	if len(c.CompensationTiers) == 0 {
		return
	}
	fmt.Println("Compensation:")
	for _, tier := range c.CompensationTiers {
		title := "(unnamed tier)"
		if v, ok := tier.Title.Get(); ok {
			title = v
		}
		fmt.Printf("  %s\n", title)
		for _, comp := range tier.Components {
			fmt.Printf("    - %s\n", componentLine(comp))
		}
	}
}

// componentLine renders a component, using the API's human-readable summary
// first and adding its type and interval when present.
func componentLine(c ashby.CompensationComponent) string {
	line := c.Summary.Value
	if line == "" {
		line = c.CompensationType.Value
	}
	var quals []string
	if c.CompensationType.Value != "" && c.CompensationType.Value != line {
		quals = append(quals, c.CompensationType.Value)
	}
	if v, ok := c.Interval.Get(); ok && v != "NONE" {
		quals = append(quals, v)
	}
	if len(quals) > 0 {
		return fmt.Sprintf("%s (%s)", line, strings.Join(quals, ", "))
	}
	return line
}
