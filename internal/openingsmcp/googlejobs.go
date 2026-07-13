package openingsmcp

import (
	"context"

	"github.com/amikai/openings-mcp/internal/provider/googlejobs"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var googleJobsSearchInputRawSchema = []byte(`{
	"type": "object",
	"properties": {
		"search_term": {
			"type": "string",
			"minLength": 1,
			"description": "Role title, skill, or technology. JobSpy appends job type, location, age, and remote phrases before searching Google Jobs."
		},
		"google_search_term": {
			"type": "string",
			"minLength": 1,
			"description": "Complete, precise Google Jobs query. When present it overrides search_term, location, job_type, remote, and hours_old rather than combining with them."
		},
		"location": {
			"type": "string",
			"description": "Location appended to the synthesized query as 'near <location>'; ignored when google_search_term is present."
		},
		"job_type": {
			"type": "string",
			"description": "Job type phrase added to the synthesized query; ignored when google_search_term is present.",
			"enum": ["fulltime", "parttime", "internship", "contract"]
		},
		"remote": {
			"type": "boolean",
			"description": "Append 'remote' to the synthesized query; ignored when google_search_term is present."
		},
		"results_wanted": {
			"type": "integer",
			"description": "Number of jobs to return. Google Jobs is cursor-paginated at about 10 per page; use a small value and narrow query. JobSpy caps this at 900.",
			"minimum": 1,
			"maximum": 900,
			"default": 15
		},
		"offset": {
			"type": "integer",
			"description": "Number of unique job URLs to skip after fetching; this is applied locally, not sent to Google.",
			"minimum": 0,
			"maximum": 900,
			"default": 0
		},
		"hours_old": {
			"type": "integer",
			"description": "Posting age converted to a coarse phrase: <=24 hours, <=3 days, <=1 week, otherwise <=1 month. Ignored when google_search_term is present.",
			"minimum": 1
		}
	},
	"anyOf": [
		{"required": ["search_term"]},
		{"required": ["google_search_term"]}
	],
	"additionalProperties": false
}`)

var googleJobsSearchInputSchema = mustSchema(googleJobsSearchInputRawSchema)

type googleJobsSearchInput struct {
	SearchTerm       string             `json:"search_term,omitempty"`
	GoogleSearchTerm string             `json:"google_search_term,omitempty"`
	Location         string             `json:"location,omitempty"`
	JobType          googlejobs.JobType `json:"job_type,omitempty"`
	Remote           bool               `json:"remote,omitempty"`
	ResultsWanted    int                `json:"results_wanted,omitempty"`
	Offset           int                `json:"offset,omitempty"`
	HoursOld         int                `json:"hours_old,omitempty"`
}

type googleJobsSearchOutput struct {
	Data    []googlejobs.Job `json:"data"`
	Warning string           `json:"warning,omitempty" jsonschema:"Non-fatal cursor or parser issue; data contains the partial results recovered before pagination stopped."`
}

func googleJobsMCPToRequest(input *googleJobsSearchInput) googlejobs.SearchRequest {
	return googlejobs.SearchRequest{
		SearchTerm:       input.SearchTerm,
		GoogleSearchTerm: input.GoogleSearchTerm,
		Location:         input.Location,
		JobType:          input.JobType,
		Remote:           input.Remote,
		ResultsWanted:    input.ResultsWanted,
		Offset:           input.Offset,
		HoursOld:         input.HoursOld,
	}
}

// RegisterGoogleJobs registers the Google Jobs aggregation search tool. Each
// result already includes the description and source posting URL, so this
// surface has no separate detail operation.
func RegisterGoogleJobs(s *mcp.Server, scraper *googlejobs.Scraper) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "google_jobs_search",
		Description: "Search Google's cross-company Google Jobs aggregation surface using behavior derived from the JobSpy upstream. Results already include descriptions and source posting URLs. Prefer a precise google_search_term; it overrides every other search filter.",
		Annotations: &mcp.ToolAnnotations{Title: "Search Google Jobs", ReadOnlyHint: true},
		InputSchema: googleJobsSearchInputSchema,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input *googleJobsSearchInput) (*mcp.CallToolResult, *googleJobsSearchOutput, error) {
		response, err := scraper.Search(ctx, googleJobsMCPToRequest(input))
		if err != nil {
			return errorResult(err), nil, nil
		}
		return nil, &googleJobsSearchOutput{Data: response.Jobs, Warning: response.Warning}, nil
	})
}
