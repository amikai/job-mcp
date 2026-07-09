package ashby

import (
	_ "embed"
	"net/http"
	"net/http/httptest"
)

// MockBoardName is the board slug used by the main fixture.
const MockBoardName = "browserbase"

// MockNullsBoardName serves jobs with null workplace fields.
const MockNullsBoardName = "weaviate"

//go:embed testdata/board_rsp.json
var mockBoardRsp []byte

//go:embed testdata/board_comp_rsp.json
var mockBoardCompRsp []byte

//go:embed testdata/board_nulls_rsp.json
var mockBoardNullsRsp []byte

// NewMockServer serves canned Ashby responses without hitting the live API.
// Compensation and unknown-board responses mirror the real API. The caller
// owns the server and must close it.
func NewMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/posting-api/job-board/"+MockBoardName, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("includeCompensation") == "true" {
			serveMockJSON(mockBoardCompRsp)(w, r)
			return
		}
		serveMockJSON(mockBoardRsp)(w, r)
	})
	mux.HandleFunc("/posting-api/job-board/"+MockNullsBoardName, serveMockJSON(mockBoardNullsRsp))
	mux.HandleFunc("/posting-api/job-board/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	})
	return httptest.NewServer(mux)
}

func serveMockJSON(data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}
