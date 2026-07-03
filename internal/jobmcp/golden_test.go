package jobmcp

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/amikai/job-mcp/internal/provider/cake"
	"github.com/amikai/job-mcp/internal/provider/google"
	"github.com/amikai/job-mcp/internal/provider/job104"
	"github.com/amikai/job-mcp/internal/provider/nvidia"
	"github.com/amikai/job-mcp/internal/provider/tsmc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool("update", false, "rewrite golden files with current output")

// TestServerGolden locks the server's LLM-facing surface — instructions plus
// the full tools/list JSON (names, descriptions, annotations, input/output
// schemas) — against testdata/server.golden.json. Any drift, including schema
// changes from a go-sdk upgrade, fails here until the golden file is
// deliberately regenerated with:
//
//	go test ./internal/jobmcp -run TestServerGolden -update
func TestServerGolden(t *testing.T) {
	ctx := context.Background()

	c104, err := job104.NewClient("https://www.104.com.tw", job104.WithClient(http.DefaultClient))
	require.NoError(t, err)
	cCake, err := cake.NewClient("https://api.cake.me", cake.WithClient(http.DefaultClient))
	require.NoError(t, err)
	cNvidia, err := nvidia.NewClient("https://nvidia.wd5.myworkdayjobs.com/wday/cxs/nvidia/NVIDIAExternalCareerSite", nvidia.WithClient(http.DefaultClient))
	require.NoError(t, err)
	cTsmc := tsmc.NewClient("https://careers.tsmc.com", http.DefaultClient)
	cGoogle := google.NewClient("https://www.google.com/about/careers/applications", http.DefaultClient)
	server := NewServer(c104, cCake, cNvidia, cTsmc, cGoogle)

	client := mcp.NewClient(&mcp.Implementation{Name: "golden", Version: "v0"}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer serverSession.Close()
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	res, err := clientSession.ListTools(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, res.NextCursor, "tools/list paginated; golden capture would be incomplete")

	snapshot := struct {
		Instructions string      `json:"instructions"`
		Tools        []*mcp.Tool `json:"tools"`
	}{
		Instructions: clientSession.InitializeResult().Instructions,
		Tools:        res.Tools,
	}
	got, err := json.MarshalIndent(snapshot, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	goldenPath := filepath.Join("testdata", "server.golden.json")
	if *updateGolden {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0o755))
		require.NoError(t, os.WriteFile(goldenPath, got, 0o644))
		return
	}

	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "golden file missing; regenerate with -update")
	assert.Equal(t, string(want), string(got))
}
