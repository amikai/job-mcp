package greenhouse

import (
	_ "embed"
	"net/http"
	"net/http/httptest"
)

//go:embed testdata/jobs_rsp.json
var mockJobsRsp []byte

//go:embed testdata/job_detail_rsp.json
var mockJobDetailRsp []byte

//go:embed testdata/job_detail_full_rsp.json
var mockJobDetailFullRsp []byte

// NewMockServer returns an httptest.Server serving canned Greenhouse Job
// Board API fixture responses captured from real boards (see
// testdata/*.sh), so tests never hit a live one. The caller owns the server
// and must Close it.
func NewMockServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/boards/safariai/jobs", serveMockJSON(mockJobsRsp))

	mux.HandleFunc("/boards/anthropic/jobs/4461450008", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("questions") == "true" {
			serveMockJSON(mockJobDetailFullRsp)(w, r)
			return
		}
		serveMockJSON(mockJobDetailRsp)(w, r)
	})

	mux.HandleFunc("/boards/doesnotexist/jobs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	mux.HandleFunc("/boards/anthropic/jobs/999999999999", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	return httptest.NewServer(mux)
}

func serveMockJSON(data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}
