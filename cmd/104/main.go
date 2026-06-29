package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/amikai/job-mcp/internal/provider/job104"
)

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

	client := job104.NewClient(http.DefaultClient)
	search, err := client.Jobs(ctx, defaultSearchParams(keyword))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	jobs := jobsForDetail(search.Data)
	details := make(map[string]*job104.JobDetailResponse, len(jobs))
	for _, job := range jobs {
		code := jobCodeFromURL(job.Link.Job)
		if code == "" {
			continue
		}
		detail, err := client.JobDetail(ctx, code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "job detail %s: %v\n", code, err)
			os.Exit(1)
		}
		details[code] = detail
	}

	writeReport(os.Stdout, keyword, search, jobs, details)
}


func defaultSearchParams(keyword string) *job104.JobsRequest {
	fullTime := 0
	return &job104.JobsRequest{
		Keyword: keyword,
		RO:      &fullTime,
	}
}

func jobCodeFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err == nil {
		raw = u.Path
	}
	raw = strings.TrimRight(raw, "/")
	parts := strings.Split(raw, "/")
	return parts[len(parts)-1]
}

func jobsForDetail(jobs []job104.Job) []job104.Job {
	limited := make([]job104.Job, 0, min(len(jobs), 10))
	for _, job := range jobs {
		if job.RemoteWorkType != 0 {
			continue
		}
		limited = append(limited, job)
		if len(limited) == 10 {
			break
		}
	}
	return limited
}

func writeReport(w io.Writer, keyword string, search *job104.JobsResponse, jobs []job104.Job, details map[string]*job104.JobDetailResponse) {
	p := search.Metadata.Pagination
	fmt.Fprintf(w, "104 Jobs Report\n")
	fmt.Fprintf(w, "Keyword: %s\n", keyword)
	fmt.Fprintf(w, "Filters: full-time, non-remote\n")
	fmt.Fprintf(w, "Found %d jobs (page %d/%d); showing %d\n\n", p.Total, p.CurrentPage, p.LastPage, len(jobs))

	for i, job := range jobs {
		code := jobCodeFromURL(job.Link.Job)
		fmt.Fprintf(w, "%d. [%s] %s\n", i+1, code, job.JobName)
		fmt.Fprintf(w, "Company: %s\n", job.CustName)
		if job.JobAddrNoDesc != "" {
			fmt.Fprintf(w, "Location: %s\n", job.JobAddrNoDesc)
		}
		if detail := details[code]; detail != nil {
			writeDetail(w, detail)
		}
		fmt.Fprintln(w)
	}
}

func writeDetail(w io.Writer, detail *job104.JobDetailResponse) {
	d := detail.Data
	jd := d.JobDetail
	if jd.Salary != "" {
		fmt.Fprintf(w, "Salary: %s\n", jd.Salary)
	}
	if d.Condition.WorkExp != "" || d.Condition.Edu != "" {
		fmt.Fprintf(w, "Experience: %s | Education: %s\n", d.Condition.WorkExp, d.Condition.Edu)
	}
	if jd.JobDescription != "" {
		fmt.Fprintf(w, "Description:\n%s\n", strings.TrimSpace(jd.JobDescription))
	}
}
