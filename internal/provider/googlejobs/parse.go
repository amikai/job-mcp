package googlejobs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const jobDataKey = "520084652"

var (
	forwardCursorPattern = regexp.MustCompile(`data-async-fc="([^"]+)"`)
	digitPattern         = regexp.MustCompile(`\d+`)
	emailPattern         = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	jobTypePatterns      = []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{name: "fulltime", pattern: regexp.MustCompile(`(?i)full\s?time`)},
		{name: "parttime", pattern: regexp.MustCompile(`(?i)part\s?time`)},
		{name: "internship", pattern: regexp.MustCompile(`(?i)internship`)},
		{name: "contract", pattern: regexp.MustCompile(`(?i)contract`)},
	}
)

func parseInitialPage(body []byte, now time.Time) ([]Job, string, error) {
	rawJobs := findInitialJobArrays(body)
	jobs := parseJobArrays(rawJobs, now)
	if len(rawJobs) > 0 && len(jobs) == 0 {
		return nil, "", errors.New("googlejobs: initial page contained job arrays but no parseable jobs")
	}
	cursor := findForwardCursor(body)
	if len(rawJobs) == 0 && cursor == "" && bytes.Contains(body, []byte("/httpservice/retry/enablejs")) {
		return nil, "", errors.New("googlejobs: Google returned its enablejs fallback instead of job data; the JobSpy HTTP protocol is currently blocked or has changed")
	}
	return jobs, cursor, nil
}

func parseNextPage(body []byte, now time.Time) ([]Job, string, error) {
	start := bytes.Index(body, []byte("[[["))
	// Google closes the outer payload with three adjacent brackets. Keep the
	// search explicit because the text envelope is not valid JSON as a whole.
	end := bytes.LastIndex(body, []byte(`]]]`))
	if start < 0 {
		return nil, "", errors.New("googlejobs: async response has no JSON payload")
	}
	if end < 0 {
		return nil, "", errors.New("googlejobs: async response has no JSON terminator")
	}

	var envelope any
	if err := decodeJSON(body[start:end+3], &envelope); err != nil {
		return nil, "", fmt.Errorf("googlejobs: decode async envelope: %w", err)
	}

	root, ok := envelope.([]any)
	if !ok || len(root) == 0 {
		return nil, "", errors.New("googlejobs: async envelope has unexpected root")
	}
	items, ok := root[0].([]any)
	if !ok {
		return nil, "", errors.New("googlejobs: async envelope has unexpected item list")
	}

	rawJobs := make([][]any, 0, len(items))
	for _, item := range items {
		pair, ok := item.([]any)
		if !ok || len(pair) < 2 {
			continue
		}
		nested, ok := pair[1].(string)
		if !ok || !strings.HasPrefix(nested, "[[[") {
			continue
		}

		var value any
		if err := decodeJSON([]byte(nested), &value); err != nil {
			continue
		}
		if info := findJobInfo(value); info != nil {
			rawJobs = append(rawJobs, info)
		}
	}

	return parseJobArrays(rawJobs, now), findForwardCursor(body), nil
}

func findInitialJobArrays(body []byte) [][]any {
	marker := []byte(`"` + jobDataKey + `":`)
	jobs := make([][]any, 0)
	for offset := 0; offset < len(body); {
		idx := bytes.Index(body[offset:], marker)
		if idx < 0 {
			break
		}
		valueStart := offset + idx + len(marker)
		var value any
		if err := decodeJSON(body[valueStart:], &value); err == nil {
			if job, ok := value.([]any); ok {
				jobs = append(jobs, job)
			}
		}
		offset = valueStart
	}
	return jobs
}

func decodeJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(target)
}

func findJobInfo(value any) []any {
	switch value := value.(type) {
	case map[string]any:
		if info, ok := value[jobDataKey].([]any); ok {
			return info
		}
		for _, nested := range value {
			if info := findJobInfo(nested); info != nil {
				return info
			}
		}
	case []any:
		for _, nested := range value {
			if info := findJobInfo(nested); info != nil {
				return info
			}
		}
	}
	return nil
}

func parseJobArrays(rawJobs [][]any, now time.Time) []Job {
	jobs := make([]Job, 0, len(rawJobs))
	for _, raw := range rawJobs {
		if job, ok := parseJob(raw, now); ok {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func parseJob(raw []any, now time.Time) (Job, bool) {
	if len(raw) <= 28 {
		return Job{}, false
	}

	title, _ := raw[0].(string)
	if title == "" {
		return Job{}, false
	}
	company, _ := raw[1].(string)
	location, _ := raw[2].(string)
	description, _ := raw[19].(string)
	id := scalarString(raw[28])
	if id == "" {
		return Job{}, false
	}

	city, state, country := splitLocation(location)
	job := Job{
		ID:          "go-" + id,
		Title:       title,
		Company:     company,
		Location:    location,
		City:        city,
		State:       state,
		Country:     country,
		URL:         nestedString(raw[3], 0, 0),
		Description: description,
		Remote:      looksRemote(description),
		Emails:      emailPattern.FindAllString(description, -1),
		JobTypes:    inferJobTypes(description),
	}

	if relative, ok := raw[12].(string); ok {
		if digits := digitPattern.FindString(relative); digits != "" {
			days, err := strconv.Atoi(digits)
			if err == nil {
				job.DatePosted = now.AddDate(0, 0, -days).Format(time.DateOnly)
			}
		}
	}
	return job, true
}

func scalarString(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case json.Number:
		return value.String()
	default:
		return ""
	}
}

func nestedString(value any, indexes ...int) string {
	current := value
	for _, index := range indexes {
		slice, ok := current.([]any)
		if !ok || index < 0 || index >= len(slice) {
			return ""
		}
		current = slice[index]
	}
	result, _ := current.(string)
	return result
}

func splitLocation(location string) (city, state, country string) {
	parts := strings.Split(location, ",")
	if len(parts) > 0 {
		city = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		state = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		country = strings.TrimSpace(parts[2])
	}
	return city, state, country
}

func looksRemote(description string) bool {
	description = strings.ToLower(description)
	return strings.Contains(description, "remote") || strings.Contains(description, "wfh")
}

func inferJobTypes(description string) []string {
	jobTypes := make([]string, 0, len(jobTypePatterns))
	for _, candidate := range jobTypePatterns {
		if candidate.pattern.MatchString(description) {
			jobTypes = append(jobTypes, candidate.name)
		}
	}
	return jobTypes
}

func findForwardCursor(body []byte) string {
	match := forwardCursorPattern.FindSubmatch(body)
	if len(match) != 2 {
		return ""
	}
	return string(match[1])
}
