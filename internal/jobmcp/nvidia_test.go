package jobmcp

import (
	"encoding/json"
	"testing"

	"github.com/amikai/job-mcp/internal/provider/nvidia"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNvidiaMCPClientServer(t *testing.T) (*mcp.ClientSession, *mcp.ServerSession) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0"}, nil)
	srv := nvidia.NewMockServer()
	t.Cleanup(srv.Close)
	client, err := nvidia.NewClient(srv.URL, nvidia.WithClient(srv.Client()))
	require.NoError(t, err)
	RegisterNvidia(server, client)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		serverSession.Close()
	})

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, nil)
	clientSession, err := mcpClient.Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		clientSession.Close()
	})
	return clientSession, serverSession
}

func TestRegisterNvidia(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0"}, nil)

	client, err := nvidia.NewClient("https://nvidia.wd5.myworkdayjobs.com/wday/cxs/nvidia/NVIDIAExternalCareerSite")
	require.NoError(t, err)
	RegisterNvidia(server, client)

	assertTools(t, server, "nvidia_search_jobs", "nvidia_get_job_detail")
}

func TestNvidiaSearchJobsE2E(t *testing.T) {
	clientSession, _ := testNvidiaMCPClientServer(t)

	res, err := clientSession.ListTools(t.Context(), nil)
	require.NoError(t, err)

	tool := findTool(res.Tools, "nvidia_search_jobs")
	require.NotNil(t, tool)

	schema, ok := tool.InputSchema.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "object", schema["type"])

	// Test calling nvidia_search_jobs tool
	callRes, err := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "nvidia_search_jobs",
		Arguments: map[string]any{
			"keyword":       "golang",
			"job_category":  "Engineering",
			"job_type":      "Regular Employee",
			"time_type":     "Full time",
			"location_type": "Remote",
			"country":       "Taiwan",
			"site":          "Taiwan, Taipei",
			"limit":         5,
			"offset":        0,
		},
	})
	require.NoError(t, err)
	assert.False(t, callRes.IsError)

	data, err := json.Marshal(callRes.StructuredContent)
	require.NoError(t, err)
	var output nvidiaSearchOutput
	err = json.Unmarshal(data, &output)
	require.NoError(t, err)

	assert.Equal(t, 27, output.Total)
	require.NotEmpty(t, output.Data)
	assert.Equal(t, "Senior Software Golang Kubernetes Engineer", output.Data[0].Title)
	assert.Equal(t, "/job/Israel-Yokneam/Senior-Software-Golang-Kubernetes-Engineer_JR2015916", output.Data[0].ExternalPath)
}

func TestNvidiaGetJobDetailE2E(t *testing.T) {
	clientSession, _ := testNvidiaMCPClientServer(t)

	res, err := clientSession.ListTools(t.Context(), nil)
	require.NoError(t, err)

	tool := findTool(res.Tools, "nvidia_get_job_detail")
	require.NotNil(t, tool)

	callRes, err := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "nvidia_get_job_detail",
		Arguments: map[string]any{
			"external_path": "/job/Israel-Yokneam/Senior-Software-Golang-Kubernetes-Engineer_JR2015916",
		},
	})
	require.NoError(t, err)
	assert.False(t, callRes.IsError)

	data, err := json.Marshal(callRes.StructuredContent)
	require.NoError(t, err)
	var output nvidiaDetailOutput
	err = json.Unmarshal(data, &output)
	require.NoError(t, err)

	assert.Equal(t, "Senior Software Golang Kubernetes Engineer", output.Title)
	assert.Contains(t, output.Description, "NVIDIA Networking is looking for an excellent Software Developer")
	assert.Equal(t, "Israel, Yokneam", output.Location)
	assert.Equal(t, []string{"Israel, Raanana", "Israel, Tel Aviv"}, output.AdditionalLocations)
	assert.Equal(t, "JR2015916", output.JobReqID)
	assert.Equal(t, "https://nvidia.wd5.myworkdayjobs.com/NVIDIAExternalCareerSite/job/Israel-Yokneam/Senior-Software-Golang-Kubernetes-Engineer_JR2015916", output.ExternalURL)
}

func TestNvidiaSearchJobsInvalidEnumE2E(t *testing.T) {
	clientSession, _ := testNvidiaMCPClientServer(t)

	// A value outside a property's enum is rejected by the SDK's
	// input-schema validation before the handler runs, as an IsError
	// tool result.
	callRes, err := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "nvidia_search_jobs",
		Arguments: map[string]any{"job_type": "valueNotInEnum"},
	})
	require.NoError(t, err)
	require.True(t, callRes.IsError)
	text, ok := callRes.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, `validating /properties/job_type: enum`)
}

func TestNvidiaHTTPToMCPResponse(t *testing.T) {
	in := nvidia.JobsResponse{
		Total: 2,
		JobPostings: []nvidia.JobSummary{
			{
				Title:         nvidia.NewOptString("t1"),
				ExternalPath:  nvidia.NewOptString("p1"),
				LocationsText: nvidia.NewOptString("l1"),
				PostedOn:      nvidia.NewOptString("d1"),
			},
			{
				Title:         nvidia.NewOptString("t2"),
				ExternalPath:  nvidia.NewOptString("p2"),
				LocationsText: nvidia.NewOptString("l2"),
				PostedOn:      nvidia.NewOptString("d2"),
			},
		},
	}
	got := nvidiaHTTPToMCPResponse(&in)

	want := &nvidiaSearchOutput{
		Total: 2,
		Data: []nvidiaJobSummary{
			{Title: "t1", ExternalPath: "p1", LocationsText: "l1", PostedOn: "d1"},
			{Title: "t2", ExternalPath: "p2", LocationsText: "l2", PostedOn: "d2"},
		},
	}
	assert.Equal(t, want, got)
}

func TestNvidiaHTTPToMCPDetail(t *testing.T) {
	in := nvidia.JobDetailResponse{
		JobPostingInfo: nvidia.JobPostingInfo{
			Title:               "t",
			JobDescription:      "<p>d</p>",
			Location:            nvidia.NewOptString("l"),
			AdditionalLocations: []string{"al"},
			PostedOn:            nvidia.NewOptString("po"),
			TimeType:            nvidia.NewOptString("tt"),
			JobReqId:            nvidia.NewOptString("id"),
			ExternalUrl:         nvidia.NewOptString("url"),
		},
	}
	got := nvidiaHTTPToMCPDetail(&in)

	want := &nvidiaDetailOutput{
		Title:               "t",
		Description:         "d",
		Location:            "l",
		AdditionalLocations: []string{"al"},
		PostedOn:            "po",
		TimeType:            "tt",
		JobReqID:            "id",
		ExternalURL:         "url",
	}
	assert.Equal(t, want, got)
}

func TestBuildNvidiaAppliedFacets(t *testing.T) {
	in := nvidiaSearchInput{
		JobCategory:  "Engineering",
		JobType:      "Regular Employee",
		TimeType:     "Full time",
		LocationType: "Remote",
		Country:      "Taiwan",
		Site:         "Taiwan, Taipei",
	}
	got, err := buildNvidiaAppliedFacets(&in)
	require.NoError(t, err)

	want := nvidia.AppliedFacets{
		JobFamilyGroup:     []nvidia.AppliedFacetsJobFamilyGroupItem{nvidia.JobCategoryIDs["Engineering"]},
		WorkerSubType:      []nvidia.AppliedFacetsWorkerSubTypeItem{nvidia.JobTypeIDs["Regular Employee"]},
		TimeType:           []nvidia.AppliedFacetsTimeTypeItem{nvidia.TimeTypeIDs["Full time"]},
		LocationHierarchy2: []nvidia.AppliedFacetsLocationHierarchy2Item{nvidia.LocationTypeIDs["Remote"]},
		LocationHierarchy1: []nvidia.AppliedFacetsLocationHierarchy1Item{nvidia.CountryIDs["Taiwan"]},
		Locations:          []nvidia.AppliedFacetsLocationsItem{nvidia.SiteIDs["Taiwan, Taipei"]},
	}
	assert.Equal(t, want, got)
}

func TestBuildNvidiaAppliedFacetsInvalidLabels(t *testing.T) {
	cases := []struct {
		name string
		in   nvidiaSearchInput
		want string
	}{
		{"job_category", nvidiaSearchInput{JobCategory: "valueNotInEnum"}, `invalid job_category "valueNotInEnum"`},
		{"job_type", nvidiaSearchInput{JobType: "valueNotInEnum"}, `invalid job_type "valueNotInEnum"`},
		{"time_type", nvidiaSearchInput{TimeType: "valueNotInEnum"}, `invalid time_type "valueNotInEnum"`},
		{"location_type", nvidiaSearchInput{LocationType: "valueNotInEnum"}, `invalid location_type "valueNotInEnum"`},
		{"country", nvidiaSearchInput{Country: "valueNotInEnum"}, `invalid country "valueNotInEnum"`},
		{"site", nvidiaSearchInput{Site: "valueNotInEnum"}, `invalid site "valueNotInEnum"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildNvidiaAppliedFacets(&tc.in)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

