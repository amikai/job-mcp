package apple

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

const (
	mockCSRFToken        = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	mockSearchKeyword    = "software engineer"
	mockFilteredKeyword  = "camera"
	mockSearchLocation   = "postLocation-TWN"
	mockFilteredLocation = "postLocation-USA"
	MockJobID            = "200624996"
	MockNotFoundJobID    = "999999999"
)

//go:embed testdata/jobs_rsp.json
var mockJobsResponse []byte

//go:embed testdata/jobs_filtered_rsp.json
var mockFilteredJobsResponse []byte

//go:embed testdata/job_detail_rsp.json
var mockJobDetailResponse []byte

//go:embed testdata/job_detail_not_found_rsp.json
var mockJobDetailNotFoundResponse []byte

// NewMockServer returns an httptest.Server that replays captured Apple Jobs
// responses, including the CSRF header and session-cookie search contract.
// The caller owns the server and must Close it.
func NewMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/CSRFToken", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("x-apple-csrf-token", mockCSRFToken)
		http.SetCookie(w, &http.Cookie{
			Name:     "jssid",
			Value:    "fixture-session",
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /api/v1/search", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("jssid")
		if err != nil || cookie.Value != "fixture-session" || r.Header.Get("x-apple-csrf-token") != mockCSRFToken {
			serveMockJSON(w, 436, mockJobDetailNotFoundResponse)
			return
		}
		fixture, ok := searchFixture(r)
		if !ok {
			serveMockJSON(w, 436, mockJobDetailNotFoundResponse)
			return
		}
		serveMockJSON(w, http.StatusOK, fixture)
	})
	mux.HandleFunc("GET /api/v1/jobDetails/{jobId}", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("locale") != "en-us" || r.PathValue("jobId") == MockNotFoundJobID {
			serveMockJSON(w, http.StatusNotFound, mockJobDetailNotFoundResponse)
			return
		}
		serveMockJSON(w, http.StatusOK, mockJobDetailResponse)
	})
	return httptest.NewTLSServer(mux)
}

type mockDateFormat struct {
	LongDate   string `json:"longDate"`
	MediumDate string `json:"mediumDate"`
}

type mockSearchFilters struct {
	Locations []string `json:"locations"`
}

type mockSearchRequest struct {
	Query   string            `json:"query"`
	Locale  string            `json:"locale"`
	Sort    string            `json:"sort"`
	Format  mockDateFormat    `json:"format"`
	Filters mockSearchFilters `json:"filters"`
	Page    int               `json:"page"`
}

func searchFixture(r *http.Request) ([]byte, bool) {
	var request mockSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, false
	}
	if !request.hasValidEnvelope() {
		return nil, false
	}
	switch {
	case request.matches(mockSearchKeyword, mockSearchLocation, "relevance", 1):
		return mockJobsResponse, true
	case request.matches(mockFilteredKeyword, mockFilteredLocation, "newest", 2):
		return mockFilteredJobsResponse, true
	default:
		return nil, false
	}
}

func (r mockSearchRequest) hasValidEnvelope() bool {
	return r.Locale == "en-us" &&
		r.Format.LongDate == "MMMM D, YYYY" &&
		r.Format.MediumDate == "MMM D, YYYY" &&
		len(r.Filters.Locations) == 1
}

func (r mockSearchRequest) matches(query, location, sort string, page int) bool {
	return r.Query == query &&
		r.Filters.Locations[0] == location &&
		r.Sort == sort &&
		r.Page == page
}

func serveMockJSON(w http.ResponseWriter, status int, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
