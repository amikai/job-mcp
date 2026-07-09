package nvidia

import (
	_ "embed"
	"net/http"
	"net/http/httptest"
)

//go:embed testdata/jobs_rsp.json
var mockJobsRsp []byte

//go:embed testdata/job_detail_rsp.json
var mockJobDetailRsp []byte

// NewMockServer serves canned NVIDIA Workday responses. The caller owns the
// server and must close it.
func NewMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/jobs", serveMockJSON(mockJobsRsp))
	mux.HandleFunc("/job/", serveMockJSON(mockJobDetailRsp))
	return httptest.NewServer(mux)
}

func serveMockJSON(data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}
