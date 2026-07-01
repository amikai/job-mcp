package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"

	"github.com/amikai/job-mcp/internal/provider/job104"
)

var areaIDs = map[string]string{
	"Taipei":     job104.AreaTaipei,
	"New Taipei": job104.AreaNewTaipei,
	"Taoyuan":    job104.AreaTaoyuan,
	"Taichung":   job104.AreaTaichung,
	"Tainan":     job104.AreaTainan,
	"Kaohsiung":  job104.AreaKaohsiung,
}

var roIDs = map[string]int{
	"Full-time": job104.ROFullTime,
	"Part-time": job104.ROPartTime,
}

var orderIDs = map[string]int{
	"Relevance": job104.OrderRelevance,
	"Newest":    job104.OrderNewest,
}

// No "no remote" choice here: the server rejects that value outright
// (confirmed live: remoteWork=0 400s). Omit --remote-work for that instead.
var remoteWorkIDs = map[string]int{
	"Partial": job104.RemoteWorkPartial,
	"Full":    job104.RemoteWorkFull,
}

// main issues a single JobsRequest built entirely from flags, then fetches
// JobDetail for every job the search returned.
func main() {
	fs := ff.NewFlagSet("104")
	var (
		timeout    = fs.DurationLong("timeout", 60*time.Second, "request timeout")
		keyword    = fs.StringLong("keyword", "", "free-text keyword search")
		area       = fs.StringEnumLong("area", usageWithChoices("Area label", areaLabels()), areaLabels()...)
		ro         = fs.StringEnumLong("ro", usageWithChoices("Job type label", intLabels(roIDs)), intLabels(roIDs)...)
		order      = fs.StringEnumLong("order", usageWithChoices("Sort order label", intLabels(orderIDs)), intLabels(orderIDs)...)
		page       = fs.IntLong("page", 0, "1-based page number (0 = unset, server default)")
		edu        = fs.StringLong("edu", "", "education code, passed through as-is")
		remoteWork = fs.StringEnumLong("remote-work", usageWithChoices("Remote work label", intLabels(remoteWorkIDs)), intLabels(remoteWorkIDs)...)
		s9         = fs.StringLong("s9", "", "comma-separated experience codes, passed through as-is")
	)
	if err := ff.Parse(fs, os.Args[1:], ff.WithEnvVarPrefix("JOB104")); err != nil {
		fmt.Fprintln(os.Stderr, ffhelp.Flags(fs))
		if errors.Is(err, ff.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "err:", err)
		os.Exit(1)
	}

	req := buildJobsRequest(*keyword, *area, *ro, *order, *edu, *remoteWork, *s9, *page)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	client := job104.NewClient(http.DefaultClient)
	search, err := client.Jobs(ctx, req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	p := search.Metadata.Pagination
	fmt.Printf("104 Jobs Report\n")
	fmt.Printf("Found %d jobs (page %d/%d); showing %d\n\n", p.Total, p.CurrentPage, p.LastPage, len(search.Data))

	for i, job := range search.Data {
		code := jobCodeFromURL(job.Link.Job)
		fmt.Printf("%d. [%s] %s\n", i+1, code, job.JobName)
		fmt.Printf("Company: %s\n", job.CustName)
		if job.JobAddrNoDesc != "" {
			fmt.Printf("Location: %s\n", job.JobAddrNoDesc)
		}
		if code == "" {
			fmt.Println()
			continue
		}
		detail, err := client.JobDetail(ctx, code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "job detail %s: %v\n", code, err)
			fmt.Println()
			continue
		}
		writeDetail(os.Stdout, detail)
		fmt.Println()
	}
}

// buildJobsRequest resolves each flag's human label to its job104 request
// value via the lookup tables above. Labels are already validated against
// the flag's enum at parse time. An empty label (flag not set) leaves that
// field at its zero value (unfiltered); page 0 leaves Page nil.
func buildJobsRequest(keyword, area, ro, order, edu, remoteWork, s9 string, page int) *job104.JobsRequest {
	req := &job104.JobsRequest{
		Keyword: keyword,
		Edu:     edu,
		S9:      s9,
	}
	if area != "" {
		req.Area = areaIDs[area]
	}
	if ro != "" {
		v := roIDs[ro]
		req.RO = &v
	}
	if order != "" {
		v := orderIDs[order]
		req.Order = &v
	}
	if page != 0 {
		req.Page = &page
	}
	if remoteWork != "" {
		v := remoteWorkIDs[remoteWork]
		req.RemoteWork = &v
	}
	return req
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

func jobCodeFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err == nil {
		raw = u.Path
	}
	raw = strings.TrimRight(raw, "/")
	parts := strings.Split(raw, "/")
	return parts[len(parts)-1]
}

// areaLabels returns the sorted keys of areaIDs, prefixed with "" so an
// ff.StringEnumLong flag can default to unset (no filter) instead of
// silently falling back to the first real label.
func areaLabels() []string {
	l := make([]string, 0, len(areaIDs)+1)
	l = append(l, "")
	for label := range areaIDs {
		l = append(l, label)
	}
	sort.Strings(l)
	return l
}

// intLabels is areaLabels for int-valued lookup tables (RO/Order/RemoteWork).
func intLabels(table map[string]int) []string {
	l := make([]string, 0, len(table)+1)
	l = append(l, "")
	for label := range table {
		l = append(l, label)
	}
	sort.Strings(l)
	return l
}

// usageWithChoices appends a comma-separated "one of: ..." list to base.
// choices is expected to include the leading "" sentinel from
// areaLabels/intLabels; it's dropped here since it's not a real choice.
func usageWithChoices(base string, choices []string) string {
	if len(choices) > 0 && choices[0] == "" {
		choices = choices[1:]
	}
	return fmt.Sprintf("%s, one of: %s", base, strings.Join(choices, " | "))
}
