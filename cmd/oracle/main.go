package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jaytaylor/html2text"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"

	"github.com/amikai/openings-mcp/internal/provider/oracle"
)

func main() {
	rootFlags := ff.NewFlagSet("oracle")
	var (
		careersURL = rootFlags.StringLong("url", "", "Oracle Candidate Experience careers URL")
		timeout    = rootFlags.DurationLong("timeout", 60*time.Second, "combined discovery and API request timeout")
		format     = rootFlags.StringEnumLong("format", "output format", "text", "json")
	)
	rootCmd := &ff.Command{
		Name:  "oracle",
		Usage: "oracle --url CAREERS-URL [FLAGS] <discover|search|facets|detail> [FLAGS]",
		Flags: rootFlags,
	}
	env := commandEnv{httpClient: http.DefaultClient, out: os.Stdout}

	discoverFlags := ff.NewFlagSet("discover").SetParent(rootFlags)
	discoverCmd := &ff.Command{
		Name:      "discover",
		Usage:     "oracle --url CAREERS-URL discover [--format text|json]",
		ShortHelp: "resolve the public site alias, internal site number, and API origin",
		Flags:     discoverFlags,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("discover takes no positional arguments, got %v", args)
			}
			return runDiscover(ctx, commonFlags{
				careersURL: *careersURL,
				timeout:    *timeout,
				format:     *format,
			}, env)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, discoverCmd)

	searchFlagsSet := ff.NewFlagSet("search").SetParent(rootFlags)
	var (
		searchKeyword = searchFlagsSet.StringLong("keyword", "", "free-text keyword search")
		searchLimit   = searchFlagsSet.IntLong("limit", 20, "page size (1-100)")
		searchOffset  = searchFlagsSet.IntLong("offset", 0, "zero-based result offset")
		searchFilters = searchFlagsSet.StringListLong("filter", "facet filter as name=id (repeatable)")
	)
	searchCmd := &ff.Command{
		Name:      "search",
		Usage:     "oracle --url CAREERS-URL search [--keyword TEXT] [--filter name=id]... [--limit N] [--offset N] [--format text|json]",
		ShortHelp: "search public requisitions with server-side keyword and facet filters",
		Flags:     searchFlagsSet,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("search takes no positional arguments, got %v (did you forget a flag name?)", args)
			}
			return runSearch(ctx, searchFlags{
				commonFlags: commonFlags{
					careersURL: *careersURL,
					timeout:    *timeout,
					format:     *format,
				},
				keyword: *searchKeyword,
				limit:   *searchLimit,
				offset:  *searchOffset,
				filters: *searchFilters,
			}, env)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, searchCmd)

	facetsFlagsSet := ff.NewFlagSet("facets").SetParent(rootFlags)
	var (
		facetsKeyword = facetsFlagsSet.StringLong("keyword", "", "narrow facet counts with a keyword")
		facetsFilters = facetsFlagsSet.StringListLong("filter", "facet filter as name=id (repeatable)")
	)
	facetsCmd := &ff.Command{
		Name:      "facets",
		Usage:     "oracle --url CAREERS-URL facets [--keyword TEXT] [--filter name=id]... [--format text|json]",
		ShortHelp: "list standard Oracle facets and their live option counts",
		Flags:     facetsFlagsSet,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("facets takes no positional arguments, got %v", args)
			}
			return runFacets(ctx, facetsFlags{
				commonFlags: commonFlags{
					careersURL: *careersURL,
					timeout:    *timeout,
					format:     *format,
				},
				keyword: *facetsKeyword,
				filters: *facetsFilters,
			}, env)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, facetsCmd)

	detailFlagsSet := ff.NewFlagSet("detail").SetParent(rootFlags)
	jobID := detailFlagsSet.StringLong("id", "", "job id from a search result")
	detailCmd := &ff.Command{
		Name:      "detail",
		Usage:     "oracle --url CAREERS-URL detail --id JOB-ID [--format text|json]",
		ShortHelp: "print one public requisition and its description sections",
		Flags:     detailFlagsSet,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("detail takes no positional arguments, got %v (did you mean --id %q?)", args, args[0])
			}
			return runDetail(ctx, detailFlags{
				commonFlags: commonFlags{
					careersURL: *careersURL,
					timeout:    *timeout,
					format:     *format,
				},
				jobID: *jobID,
			}, env)
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, detailCmd)

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
		fmt.Fprintln(os.Stderr, "err: a subcommand (discover, search, facets, or detail) is required")
		os.Exit(1)
	}
	if err := rootCmd.Run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "err:", err)
		os.Exit(1)
	}
}

type commandEnv struct {
	httpClient *http.Client
	out        io.Writer
}

type commonFlags struct {
	careersURL string
	timeout    time.Duration
	format     string
}

func (f commonFlags) context(parent context.Context) (context.Context, context.CancelFunc, error) {
	if strings.TrimSpace(f.careersURL) == "" {
		return nil, nil, errors.New("--url is required")
	}
	if f.timeout <= 0 {
		return nil, nil, fmt.Errorf("--timeout must be greater than zero, got %s", f.timeout)
	}
	ctx, cancel := context.WithTimeout(parent, f.timeout)
	return ctx, cancel, nil
}

func runDiscover(parent context.Context, flags commonFlags, env commandEnv) error {
	ctx, cancel, err := flags.context(parent)
	if err != nil {
		return err
	}
	defer cancel()

	site, err := oracle.DiscoverSite(ctx, flags.careersURL, env.httpClient)
	if err != nil {
		return err
	}
	if flags.format == "json" {
		return writeJSON(env.out, site)
	}
	fmt.Fprintf(env.out, "Careers URL: %s\n", site.CareersURL)
	fmt.Fprintf(env.out, "API Base URL: %s\n", site.APIBaseURL)
	fmt.Fprintf(env.out, "Site: %s\n", site.Site)
	fmt.Fprintf(env.out, "Site Number: %s\n", site.SiteNumber)
	fmt.Fprintf(env.out, "Language: %s\n", site.Language)
	return nil
}

type searchFlags struct {
	commonFlags
	keyword string
	limit   int
	offset  int
	filters []string
}

func runSearch(parent context.Context, flags searchFlags, env commandEnv) error {
	filters, err := parseFilters(flags.filters)
	if err != nil {
		return err
	}
	ctx, cancel, err := flags.context(parent)
	if err != nil {
		return err
	}
	defer cancel()

	client, err := oracle.DiscoverClient(ctx, flags.careersURL, env.httpClient)
	if err != nil {
		return err
	}
	result, err := client.Search(ctx, oracle.SearchRequest{
		Keyword: flags.keyword,
		Limit:   flags.limit,
		Offset:  flags.offset,
		Filters: filters,
	})
	if err != nil {
		return err
	}
	if flags.format == "json" {
		return writeJSON(env.out, result)
	}

	site := client.Site()
	fmt.Fprintf(env.out, "Oracle Recruiting Cloud Jobs (site: %s, site number: %s)\n", site.Site, site.SiteNumber)
	fmt.Fprintf(env.out, "Found %d jobs; showing %d\n\n", result.Total, len(result.Jobs))
	for i, job := range result.Jobs {
		fmt.Fprintf(env.out, "%d. %s\n", i+1, job.Title)
		if job.PrimaryLocation != "" {
			fmt.Fprintf(env.out, "Location: %s\n", job.PrimaryLocation)
		}
		if job.WorkplaceType != "" {
			fmt.Fprintf(env.out, "Workplace: %s\n", job.WorkplaceType)
		}
		if !job.PostedAt.IsZero() {
			fmt.Fprintf(env.out, "Posted: %s\n", job.PostedAt.Format("2006-01-02"))
		}
		fmt.Fprintf(env.out, "ID: %s\n", job.ID)
		fmt.Fprintf(env.out, "URL: %s\n\n", job.URL)
	}
	return nil
}

type facetsFlags struct {
	commonFlags
	keyword string
	filters []string
}

func runFacets(parent context.Context, flags facetsFlags, env commandEnv) error {
	filters, err := parseFilters(flags.filters)
	if err != nil {
		return err
	}
	ctx, cancel, err := flags.context(parent)
	if err != nil {
		return err
	}
	defer cancel()

	client, err := oracle.DiscoverClient(ctx, flags.careersURL, env.httpClient)
	if err != nil {
		return err
	}
	result, err := client.Search(ctx, oracle.SearchRequest{
		Keyword: flags.keyword,
		Limit:   1,
		Facets:  oracle.AllFacets(),
		Filters: filters,
	})
	if err != nil {
		return err
	}
	if flags.format == "json" {
		return writeJSON(env.out, result.Facets)
	}

	for _, facet := range oracle.AllFacets() {
		options := result.Facets[facet]
		if len(options) == 0 {
			continue
		}
		fmt.Fprintf(env.out, "%s:\n", facet)
		for _, option := range options {
			fmt.Fprintf(env.out, "  %s (%s): %d\n", option.Name, option.ID, option.Count)
		}
	}
	return nil
}

type detailFlags struct {
	commonFlags
	jobID string
}

func runDetail(parent context.Context, flags detailFlags, env commandEnv) error {
	if strings.TrimSpace(flags.jobID) == "" {
		return errors.New("--id is required (take it from a search result's ID)")
	}
	ctx, cancel, err := flags.context(parent)
	if err != nil {
		return err
	}
	defer cancel()

	client, err := oracle.DiscoverClient(ctx, flags.careersURL, env.httpClient)
	if err != nil {
		return err
	}
	job, err := client.Detail(ctx, flags.jobID)
	if err != nil {
		return err
	}
	if flags.format == "json" {
		return writeJSON(env.out, job)
	}

	fmt.Fprintln(env.out, job.Title)
	if job.PrimaryLocation != "" {
		fmt.Fprintf(env.out, "Location: %s\n", job.PrimaryLocation)
	}
	if job.WorkplaceType != "" {
		fmt.Fprintf(env.out, "Workplace: %s\n", job.WorkplaceType)
	}
	if !job.PostedAt.IsZero() {
		fmt.Fprintf(env.out, "Posted: %s\n", job.PostedAt.Format("2006-01-02"))
	}
	fmt.Fprintf(env.out, "ID: %s\n", job.ID)
	fmt.Fprintf(env.out, "URL: %s\n", job.URL)

	printHTMLSection(env.out, "Description", job.DescriptionHTML)
	printHTMLSection(env.out, "Company", job.CorporateDescriptionHTML)
	printHTMLSection(env.out, "Responsibilities", job.ResponsibilitiesHTML)
	printHTMLSection(env.out, "Qualifications", job.QualificationsHTML)
	return nil
}

func parseFilters(values []string) (map[oracle.Facet][]string, error) {
	filters := make(map[oracle.Facet][]string)
	for _, raw := range values {
		name, value, ok := strings.Cut(raw, "=")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if !ok || name == "" || value == "" {
			return nil, fmt.Errorf("--filter %q must be name=id", raw)
		}
		facet, err := oracle.ParseFacet(name)
		if err != nil {
			return nil, fmt.Errorf("--filter %q: %w", raw, err)
		}
		filters[facet] = append(filters[facet], value)
	}
	return filters, nil
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printHTMLSection(w io.Writer, title, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	text, err := html2text.FromString(value, html2text.Options{})
	if err != nil {
		text = value
	}
	fmt.Fprintf(w, "\n%s:\n%s\n", title, strings.TrimSpace(text))
}
