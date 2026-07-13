package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"

	"github.com/amikai/openings-mcp/internal/provider/googlejobs"
)

const defaultBaseURL = "https://www.google.com"

func main() {
	rootFlags := ff.NewFlagSet("googlejobs")
	var (
		baseURL = rootFlags.StringLong("base-url", defaultBaseURL, "Google Search base URL")
		timeout = rootFlags.DurationLong("timeout", 60*time.Second, "request timeout")
		format  = rootFlags.StringEnumLong("format", "output format", "text", "json")
	)
	rootCmd := &ff.Command{
		Name:  "googlejobs",
		Usage: "googlejobs [FLAGS] search [FLAGS]",
		Flags: rootFlags,
	}

	searchFlags := ff.NewFlagSet("search").SetParent(rootFlags)
	var (
		searchTerm       = searchFlags.StringLong("search-term", "", "role, skill, or technology; JobSpy appends the remaining filters")
		googleSearchTerm = searchFlags.StringLong("google-search-term", "", "complete Google Jobs query; overrides every synthesized search filter")
		location         = searchFlags.StringLong("location", "", "location appended as 'near <location>'")
		jobType          = searchFlags.StringEnumLong("job-type", "job type phrase", "", string(googlejobs.JobTypeFullTime), string(googlejobs.JobTypePartTime), string(googlejobs.JobTypeInternship), string(googlejobs.JobTypeContract))
		remote           = searchFlags.BoolLong("remote", "append the remote keyword")
		resultsWanted    = searchFlags.IntLong("results", googlejobs.DefaultResultsWanted, "number of results (capped at 900 by the provider)")
		offset           = searchFlags.IntLong("offset", 0, "number of unique results to skip locally (0-900)")
		hoursOld         = searchFlags.IntLong("hours-old", 0, "posting age converted to JobSpy's coarse query phrases (0 disables)")
	)
	searchCmd := &ff.Command{
		Name:      "search",
		Usage:     "googlejobs search (--search-term TEXT | --google-search-term QUERY) [--location PLACE] [--job-type TYPE] [--remote] [--hours-old N] [--results N] [--offset N] [--format text|json]",
		ShortHelp: "search the Google Jobs aggregation surface using JobSpy behavior",
		Flags:     searchFlags,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("search takes no positional arguments, got %v (did you forget a flag name?)", args)
			}
			return runSearch(ctx, cliSearchOptions{
				baseURL:          *baseURL,
				timeout:          *timeout,
				format:           *format,
				searchTerm:       *searchTerm,
				googleSearchTerm: *googleSearchTerm,
				location:         *location,
				jobType:          googlejobs.JobType(*jobType),
				remote:           *remote,
				resultsWanted:    *resultsWanted,
				offset:           *offset,
				hoursOld:         *hoursOld,
			})
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, searchCmd)

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
		fmt.Fprintln(os.Stderr, "err: the search subcommand is required")
		os.Exit(1)
	}
	if err := rootCmd.Run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "err:", err)
		os.Exit(1)
	}
}

type cliSearchOptions struct {
	baseURL          string
	timeout          time.Duration
	format           string
	searchTerm       string
	googleSearchTerm string
	location         string
	jobType          googlejobs.JobType
	remote           bool
	resultsWanted    int
	offset           int
	hoursOld         int
}

func runSearch(ctx context.Context, opts cliSearchOptions) error {
	if opts.timeout <= 0 {
		return errors.New("--timeout must be positive")
	}
	ctx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()

	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("create cookie jar: %w", err)
	}
	httpClient := &http.Client{
		Timeout:   opts.timeout,
		Jar:       jar,
		Transport: googlejobs.BrowserTransport{},
	}
	client, err := googlejobs.NewClient(opts.baseURL, googlejobs.WithClient(httpClient))
	if err != nil {
		return fmt.Errorf("create generated client: %w", err)
	}
	scraper, err := googlejobs.NewScraper(client)
	if err != nil {
		return err
	}
	response, err := scraper.Search(ctx, googlejobs.SearchRequest{
		SearchTerm:       opts.searchTerm,
		GoogleSearchTerm: opts.googleSearchTerm,
		Location:         opts.location,
		JobType:          opts.jobType,
		Remote:           opts.remote,
		ResultsWanted:    opts.resultsWanted,
		Offset:           opts.offset,
		HoursOld:         opts.hoursOld,
	})
	if err != nil {
		return err
	}
	return writeResponse(response, opts.format)
}

func writeResponse(response *googlejobs.SearchResponse, format string) error {
	if format == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(response)
	}

	fmt.Printf("Google Jobs Report\nFound %d jobs\n", len(response.Jobs))
	if response.Warning != "" {
		fmt.Printf("Warning: %s\n", response.Warning)
	}
	fmt.Println()
	for i, job := range response.Jobs {
		fmt.Printf("%d. [%s] %s\n", i+1, job.ID, job.Title)
		if job.Company != "" {
			fmt.Printf("Company: %s\n", job.Company)
		}
		if job.Location != "" {
			fmt.Printf("Location: %s\n", job.Location)
		}
		if job.DatePosted != "" {
			fmt.Printf("Posted: %s\n", job.DatePosted)
		}
		if job.URL != "" {
			fmt.Printf("URL: %s\n", job.URL)
		}
		if len(job.JobTypes) > 0 {
			fmt.Printf("Job types: %v\n", job.JobTypes)
		}
		if job.Remote {
			fmt.Println("Looks remote")
		}
		if job.Description != "" {
			fmt.Printf("Description: %s\n", job.Description)
		}
		fmt.Println()
	}
	return nil
}
