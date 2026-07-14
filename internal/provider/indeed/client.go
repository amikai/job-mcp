package indeed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// apiKey is the static key python-jobspy's constant.py embeds and every
// installation of it reuses verbatim; see openapi.yaml's Key Behaviors on
// why this is not a per-caller secret to replace.
const apiKey = "161092c2017b5bbab13edb12461a62d5a833871e7cad6d9d475304573de67ac8"

const userAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Indeed App 193.1"
const appInfo = "appv=193.1; appid=com.indeed.jobsearch; osv=16.6.1; os=ios; dtype=phone"

type Client struct {
	httpClient *http.Client
	apiURL     string // GraphQL endpoint, e.g. https://apis.indeed.com/graphql
}

// NewClient builds a Client against apiURL (the GraphQL endpoint). When
// httpClient is nil, http.DefaultClient is used.
func NewClient(apiURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient, apiURL: apiURL}
}

type graphqlRequestBody struct {
	Query string `json:"query"`
}

// resolveCountry looks up r's Country (defaulting to DefaultCountryName),
// reporting an error for a name CountryByName doesn't recognize.
func resolveCountry(name string) (Country, error) {
	if name == "" {
		name = DefaultCountryName
	}
	c, ok := CountryByName(name)
	if !ok {
		return Country{}, fmt.Errorf("unknown country %q", name)
	}
	return c, nil
}

// siteBaseURL builds the country-specific Indeed website's base URL, used
// for job/company links (distinct from the GraphQL API host, which is the
// same apis.indeed.com regardless of country).
func siteBaseURL(c Country) string {
	return "https://" + c.Domain + ".indeed.com"
}

// Jobs searches jobSearch with r's criteria.
func (c *Client) Jobs(ctx context.Context, r *JobsRequest) (*JobsResponse, error) {
	country, err := resolveCountry(r.Country)
	if err != nil {
		return nil, err
	}
	var wire wireSearchResponse
	if err := c.do(ctx, country, searchQuery(r), &wire); err != nil {
		return nil, fmt.Errorf("search jobs: %w", err)
	}
	if wire.Data.JobSearch == nil {
		return &JobsResponse{}, nil
	}
	base := siteBaseURL(country)
	jobs := make([]Job, 0, len(wire.Data.JobSearch.Results))
	for _, result := range wire.Data.JobSearch.Results {
		jobs = append(jobs, jobFromWire(result.Job, base))
	}
	resp := &JobsResponse{Jobs: jobs}
	if wire.Data.JobSearch.PageInfo.NextCursor != nil {
		resp.NextCursor = *wire.Data.JobSearch.PageInfo.NextCursor
	}
	return resp, nil
}

// JobDetail looks up one job by its key (Job.Key from a prior Jobs call).
// A key with no matching job (removed, expired, or never valid) returns
// (nil, nil) rather than an error — see openapi.yaml's Key Behaviors on
// jobData's empty-list-not-404 shape.
func (c *Client) JobDetail(ctx context.Context, country, jobKey string) (*JobDetail, error) {
	if jobKey == "" {
		return nil, errors.New("empty job key")
	}
	resolved, err := resolveCountry(country)
	if err != nil {
		return nil, err
	}
	var wire wireDetailResponse
	if err := c.do(ctx, resolved, detailQuery(jobKey), &wire); err != nil {
		return nil, fmt.Errorf("job detail %q: %w", jobKey, err)
	}
	if wire.Data.JobData == nil || len(wire.Data.JobData.Results) == 0 {
		return nil, nil
	}
	detail := jobDetailFromWire(wire.Data.JobData.Results[0].Job, siteBaseURL(resolved))
	return &detail, nil
}

func (c *Client) do(ctx context.Context, country Country, query string, out interface {
	graphqlErrors() []wireGraphQLError
}) error {
	body, err := json.Marshal(graphqlRequestBody{Query: query})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json")
	req.Header.Set("indeed-locale", "en-US")
	req.Header.Set("user-agent", userAgent)
	req.Header.Set("indeed-app-info", appInfo)
	req.Header.Set("indeed-api-key", apiKey)
	req.Header.Set("indeed-co", country.APICode)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if errs := out.graphqlErrors(); len(errs) > 0 {
		return fmt.Errorf("graphql error: %s", errs[0].Message)
	}
	return nil
}

func (r *wireSearchResponse) graphqlErrors() []wireGraphQLError { return r.Errors }
func (r *wireDetailResponse) graphqlErrors() []wireGraphQLError { return r.Errors }
