package indeed

import (
	"fmt"
	"strconv"
	"strings"
)

const searchFieldSet = `
	trackingKey
	job {
		key
		title
		datePublished
		location { countryCode admin1Code city formatted { long } }
		compensation {
			estimated { currencyCode baseSalary { unitOfWork range { ... on Range { min max } } } }
			baseSalary { unitOfWork range { ... on Range { min max } } }
			currencyCode
		}
		attributes { key label }
		employer { relativeCompanyPageUrl name }
	}
`

const detailFieldSet = `
	job {
		key
		title
		datePublished
		description { html }
		location { countryName countryCode admin1Code city postalCode streetAddress formatted { short long } }
		compensation {
			estimated { currencyCode baseSalary { unitOfWork range { ... on Range { min max } } } }
			baseSalary { unitOfWork range { ... on Range { min max } } }
			currencyCode
		}
		attributes { key label }
		employer {
			relativeCompanyPageUrl
			name
			dossier {
				employerDetails { addresses industry employeesLocalizedLabel revenueLocalizedLabel briefDescription }
				images { squareLogoUrl }
				links { corporateWebsite }
			}
		}
		recruit { viewJobUrl detailedSalary workSchedule }
	}
`

// searchQuery builds the jobSearch GraphQL query document for r. Filters are
// mutually exclusive in the shape the real API was exercised against — see
// openapi.yaml's Key Behaviors — so at most one of HoursOld, EasyApply, or
// JobType/Remote is sent, in that precedence order, mirroring python-jobspy's
// _build_filters.
func searchQuery(r *JobsRequest) string {
	var b strings.Builder
	b.WriteString("query GetJobData { jobSearch(")
	if r.Keywords != "" {
		fmt.Fprintf(&b, "what: %s, ", graphqlString(r.Keywords))
	}
	if r.Location != "" {
		radius := r.RadiusMiles
		if radius == 0 {
			radius = 25
		}
		fmt.Fprintf(&b, "location: {where: %s, radius: %d, radiusUnit: MILES}, ", graphqlString(r.Location), radius)
	}
	limit := r.Limit
	if limit == 0 {
		limit = 25
	}
	fmt.Fprintf(&b, "limit: %d, sort: RELEVANCE, ", limit)
	if r.Cursor != "" {
		fmt.Fprintf(&b, "cursor: %s, ", graphqlString(r.Cursor))
	}
	if filter := searchFilters(r); filter != "" {
		b.WriteString(filter)
	}
	b.WriteString(") { pageInfo { nextCursor } results {")
	b.WriteString(searchFieldSet)
	b.WriteString("} } }")
	return b.String()
}

func searchFilters(r *JobsRequest) string {
	switch {
	case r.HoursOld > 0:
		return fmt.Sprintf(`filters: { date: { field: "dateOnIndeed", start: "%dh" } }, `, r.HoursOld)
	case r.EasyApply:
		return `filters: { keyword: { field: "indeedApplyScope", keys: ["DESKTOP"] } }, `
	case r.JobType != "" || r.Remote:
		var keys []string
		if r.JobType != "" {
			keys = append(keys, r.JobType)
		}
		if r.Remote {
			keys = append(keys, remoteAttributeKey)
		}
		quoted := make([]string, len(keys))
		for i, k := range keys {
			quoted[i] = graphqlString(k)
		}
		return fmt.Sprintf(`filters: { composite: { filters: [{ keyword: { field: "attributes", keys: [%s] } }] } }, `, strings.Join(quoted, ", "))
	default:
		return ""
	}
}

// detailQuery builds the jobData GraphQL query document for one or more job
// keys. jobData takes a list per call (see openapi.yaml), but this client
// only ever calls it with a single key.
func detailQuery(jobKey string) string {
	return "query GetJobDetail { jobData(jobKeys: [" + graphqlString(jobKey) + "]) { results {" + detailFieldSet + "} } }"
}

// graphqlString quotes and escapes s for embedding in a GraphQL query
// document string (the API takes the query as a JSON string, not a
// structured request, so arguments are interpolated as text).
func graphqlString(s string) string {
	return strconv.Quote(s)
}
