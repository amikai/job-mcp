package tsmc

import (
	_ "embed"
	"net/http"
	"net/http/httptest"
)

//go:embed testdata/jobs_rsp.html
var mockJobsRsp []byte

//go:embed testdata/job_detail_rsp.html
var mockJobDetailRsp []byte

// NewMockServer serves canned TSMC career-site responses. The caller owns the
// server and must close it.
func NewMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/zh_TW/careers/SearchJobs/", serveMockHTML(mockJobsRsp))
	mux.HandleFunc("/zh_TW/careers/JobDetail", serveMockHTML(mockJobDetailRsp))
	return httptest.NewServer(mux)
}

func serveMockHTML(data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	}
}
