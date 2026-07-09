package job104

import (
	_ "embed"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

//go:embed testdata/jobs_rsp.json
var mockJobsRsp []byte

//go:embed testdata/job_detail_rsp.json
var mockJobDetailRsp []byte

// Response captured from 104's company-keyword mode (issue #94).
//
//go:embed testdata/issue_94_rsp.json
var mockCompanyKeywordRsp []byte

// MockErrorKeyword and MockNotFoundJobCode exercise the API's JSON 500 and 404
// error responses.
const (
	MockErrorKeyword    = "mock-500"
	MockNotFoundJobCode = "mock-404"
)

// MockCompanyKeyword triggers 104's pagination-less company-keyword response
// unless excludeCompanyKeyword=true is sent.
const MockCompanyKeyword = "聯發科"

// NewMockServer serves canned 104 responses. The caller owns the server and
// must close it.
func NewMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/jobs/search/api/jobs", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("keyword") == MockErrorKeyword {
			serveMockError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if r.URL.Query().Get("keyword") == MockCompanyKeyword && r.URL.Query().Get("excludeCompanyKeyword") != "true" {
			serveMockJSON(mockCompanyKeywordRsp)(w, r)
			return
		}
		serveMockJSON(mockJobsRsp)(w, r)
	})
	mux.HandleFunc("/job/ajax/content/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/"+MockNotFoundJobCode) {
			serveMockError(w, http.StatusNotFound, "job not found")
			return
		}
		serveMockJSON(mockJobDetailRsp)(w, r)
	})
	return httptest.NewServer(mux)
}

func serveMockJSON(data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}

func serveMockError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"message":%q}`, msg)
}
