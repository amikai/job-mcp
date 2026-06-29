package main

import (
	"bufio"
	"context"
	"fmt"
	"html"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	cake "github.com/amikai/job-mcp/internal/provider/cake"
)

var tagRE = regexp.MustCompile(`<[^>]+>`)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	keyword := strings.TrimSpace(scanner.Text())
	if keyword == "" {
		fmt.Fprintln(os.Stderr, "keyword is required")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := cake.NewClient("https://api.cake.me")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	req := defaultSearchRequest(keyword)
	searchRes, err := client.SearchJobs(ctx, &req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	search, ok := searchRes.(*cake.JobSearchResponse)
	if !ok {
		fmt.Fprintf(os.Stderr, "search returned %T\n", searchRes)
		os.Exit(1)
	}

	jobs := jobsForDetail(search.Data)
	details := make(map[string]*cake.JobDetail, len(jobs))
	for _, job := range jobs {
		detailRes, err := client.GetJobDetail(ctx, cake.GetJobDetailParams{Path: job.Path})
		if err != nil {
			fmt.Fprintf(os.Stderr, "job detail %s: %v\n", job.Path, err)
			os.Exit(1)
		}
		detail, ok := detailRes.(*cake.JobDetail)
		if !ok {
			fmt.Fprintf(os.Stderr, "job detail %s returned %T\n", job.Path, detailRes)
			os.Exit(1)
		}
		details[job.Path] = detail
	}

	writeReport(os.Stdout, keyword, search, jobs, details)
}


func defaultSearchRequest(keyword string) cake.JobSearchRequest {
	return cake.JobSearchRequest{
		Query:  keyword,
		SortBy: cake.JobSearchRequestSortByPopularity,
		Filters: cake.JobSearchRequestFilters{
			"job_types": []byte(`["full_time"]`),
			"remote":    []byte(`["no_remote_work"]`),
		},
	}
}

func jobsForDetail(jobs []cake.JobSearchItem) []cake.JobSearchItem {
	if len(jobs) > 10 {
		return jobs[:10]
	}
	return jobs
}

func writeReport(w io.Writer, keyword string, search *cake.JobSearchResponse, jobs []cake.JobSearchItem, details map[string]*cake.JobDetail) {
	fmt.Fprintf(w, "Cake Jobs Report\n")
	fmt.Fprintf(w, "Keyword: %s\n", keyword)
	fmt.Fprintf(w, "Filters: full-time, non-remote\n")
	fmt.Fprintf(w, "Found %d jobs (page %d/%d); showing %d\n\n", search.TotalEntries, search.CurrentPage, search.TotalPages, len(jobs))

	for i, job := range jobs {
		fmt.Fprintf(w, "%d. [%s] %s\n", i+1, job.Path, job.Title)
		if detail := details[job.Path]; detail != nil {
			writeDetail(w, detail)
		}
		fmt.Fprintln(w)
	}
}

func writeDetail(w io.Writer, detail *cake.JobDetail) {
	fmt.Fprintf(w, "URL: https://www.cake.me/companies/%s/jobs/%s\n", detail.PagePath, detail.Path)
	description := plainText(detail.Description)
	if description != "" {
		fmt.Fprintf(w, "Description:\n%s\n", description)
	}
	requirements := plainText(detail.Requirements)
	if requirements != "" {
		fmt.Fprintf(w, "Requirements: %s\n", requirements)
	}
}

func plainText(s string) string {
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = tagRE.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	lines := strings.Fields(s)
	return strings.Join(lines, " ")
}
