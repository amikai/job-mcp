package openingsmcp

import (
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amikai/openings-mcp/internal/provider/meta"
)

const (
	metaTestServerName = "test"
	metaTestClientName = "test-client"
	metaTestJobIDKey   = "job_id"
	metaTestJobTitle   = "Instagram Product Designer, Brand-in-Product"
)

func testMetaMCPClientServer(t *testing.T) *mcp.ClientSession {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: metaTestServerName, Version: "v0"}, nil)
	mock := meta.NewMockServer()
	t.Cleanup(mock.Close)
	RegisterMeta(server, meta.NewClient(mock.URL, mock.Client()))

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: metaTestClientName, Version: "v0"}, nil)
	clientSession, err := mcpClient.Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })
	return clientSession
}

func TestRegisterMeta(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: metaTestServerName, Version: "v0"}, nil)
	RegisterMeta(server, meta.NewClient("https://www.metacareers.com", nil))
	assertTools(t, server, metaSearchToolName, metaDetailToolName)
}

func TestMetaSearchJobsE2E(t *testing.T) {
	client := testMetaMCPClientServer(t)
	result, err := client.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      metaSearchToolName,
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	data, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	var output metaSearchOutput
	require.NoError(t, json.Unmarshal(data, &output))
	assert.Equal(t, 648, output.Total)
	require.Len(t, output.Data, 648)
	assert.Equal(t, metaJobSummary{
		JobID:     meta.MockJobID,
		URL:       "https://www.metacareers.com/jobs/1063741453022215/",
		Title:     metaTestJobTitle,
		Locations: []string{"Menlo Park, CA", "New York, NY", "San Francisco, CA"},
		Teams:     []string{"Design & User Experience", "Creative"},
		SubTeams:  []string{"Design"},
	}, output.Data[0])
}

func TestMetaSearchJobsFilteredE2E(t *testing.T) {
	client := testMetaMCPClientServer(t)
	result, err := client.CallTool(t.Context(), &mcp.CallToolParams{
		Name: metaSearchToolName,
		Arguments: map[string]any{
			"keyword": "engineer",
			"offices": []string{"Singapore"},
		},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	data, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	var output metaSearchOutput
	require.NoError(t, json.Unmarshal(data, &output))
	require.NotEmpty(t, output.Data)
	for _, job := range output.Data {
		assert.Contains(t, job.Locations, "Singapore")
	}
}

func TestMetaGetJobDetailE2E(t *testing.T) {
	client := testMetaMCPClientServer(t)
	result, err := client.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      metaDetailToolName,
		Arguments: map[string]any{metaTestJobIDKey: meta.MockJobID},
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	data, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	var output metaDetailOutput
	require.NoError(t, json.Unmarshal(data, &output))
	assert.Equal(t, meta.MockJobID, output.JobID)
	assert.Equal(t, metaTestJobTitle, output.Title)
	assert.Equal(t, []string{"Menlo Park, CA", "New York, NY", "San Francisco, CA"}, output.Locations)
	assert.Contains(t, output.Description, "Product Designer")
	assert.NotContains(t, output.Description, "<span>")
	assert.NotEmpty(t, output.Responsibilities)
	assert.NotEmpty(t, output.MinimumQualifications)
	assert.NotEmpty(t, output.PreferredQualifications)
	require.Len(t, output.Compensation, 1)
	assert.Equal(t, metaCompensation{
		CountryCode: "US",
		Minimum:     "$201,000/year",
		Maximum:     "$278,000/year",
		HasBonus:    true,
		HasEquity:   true,
	}, output.Compensation[0])
}

func TestMetaGetJobDetailNotFoundE2E(t *testing.T) {
	client := testMetaMCPClientServer(t)
	result, err := client.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      metaDetailToolName,
		Arguments: map[string]any{metaTestJobIDKey: meta.MockNotFoundJobID},
	})
	require.NoError(t, err)
	require.True(t, result.IsError)
	content, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, content.Text, "job not found")
}
