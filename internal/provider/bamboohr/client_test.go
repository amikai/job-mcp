package bamboohr

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, srvURL string) *Client {
	t.Helper()
	client, err := NewClient(srvURL)
	require.NoError(t, err)
	return client
}

func TestListJobs(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	res, err := newTestClient(t, srv.URL).ListJobs(t.Context())
	require.NoError(t, err)

	list, ok := res.(*ListResponse)
	require.True(t, ok, "expected *ListResponse, got %T", res)

	assert.Equal(t, 5, list.Meta.TotalCount)
	require.Len(t, list.Result, 5)

	// Row 0 has no department: both department fields are null.
	nulls := list.Result[0]
	assert.Equal(t, MockNullsJobID, nulls.ID)
	assert.Equal(t, "Talent Community @ Giatec", nulls.JobOpeningName)
	assert.Equal(t, NilString{Null: true}, nulls.DepartmentId)
	assert.Equal(t, NilString{Null: true}, nulls.DepartmentLabel)
	assert.Equal(t, "Full-Time", nulls.EmploymentStatusLabel)
	assert.Equal(t, NilString{Value: "Ottawa"}, nulls.Location.City)
	assert.Equal(t, NilString{Value: "Ontario"}, nulls.Location.State)
	assert.Equal(t, NilString{Null: true}, nulls.AtsLocation.Country)
	assert.Equal(t, NilBool{Null: true}, nulls.IsRemote)
	assert.Equal(t, NilString{Value: "2"}, nulls.LocationType)

	job := list.Result[2]
	assert.Equal(t, "292", job.ID)
	assert.Equal(t, "IT Operations Lead", job.JobOpeningName)
	assert.Equal(t, NilString{Value: "18811"}, job.DepartmentId)
	assert.Equal(t, NilString{Value: "Software Development - SmartMix"}, job.DepartmentLabel)
}

// TestListJobsVariety covers the value shapes the happy board doesn't: a
// null location.city whose real location lives in atsLocation, and the
// remote locationType.
func TestListJobsVariety(t *testing.T) {
	srv := NewVarietyMockServer()
	defer srv.Close()

	res, err := newTestClient(t, srv.URL).ListJobs(t.Context())
	require.NoError(t, err)

	list, ok := res.(*ListResponse)
	require.True(t, ok, "expected *ListResponse, got %T", res)
	assert.Equal(t, 21, list.Meta.TotalCount)
	require.Len(t, list.Result, 21)

	var job ListJob
	for _, j := range list.Result {
		if j.ID == "339" {
			job = j
		}
	}
	require.Equal(t, "339", job.ID)
	assert.Equal(t, "Tugboat Engineer", job.JobOpeningName)
	assert.Equal(t, NilString{Null: true}, job.Location.City)
	assert.Equal(t, NilString{Null: true}, job.Location.State)
	assert.Equal(t, NilString{Value: "United States"}, job.AtsLocation.Country)
	assert.Equal(t, NilString{Value: "California"}, job.AtsLocation.State)
	assert.Equal(t, NilString{Null: true}, job.AtsLocation.Province)
	assert.Equal(t, NilString{Value: "Long Beach"}, job.AtsLocation.City)
	assert.Equal(t, NilString{Value: "1"}, job.LocationType)
}

func TestListJobsEmpty(t *testing.T) {
	srv := NewEmptyMockServer()
	defer srv.Close()

	res, err := newTestClient(t, srv.URL).ListJobs(t.Context())
	require.NoError(t, err)

	list, ok := res.(*ListResponse)
	require.True(t, ok, "expected *ListResponse, got %T", res)
	assert.Equal(t, 0, list.Meta.TotalCount)
	assert.Empty(t, list.Result)
}

// TestListJobsUnknownTenant exercises the 302-to-marketing-site behavior an
// unknown subdomain produces. The client must not follow the redirect, or
// it lands on an HTML page the decoder rejects.
func TestListJobsUnknownTenant(t *testing.T) {
	srv := NewRedirectMockServer()
	defer srv.Close()

	hc := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	client, err := NewClient(srv.URL, WithClient(hc))
	require.NoError(t, err)

	res, err := client.ListJobs(t.Context())
	require.NoError(t, err)
	_, ok := res.(*ListJobsFound)
	require.True(t, ok, "expected *ListJobsFound, got %T", res)

	dres, err := client.GetJobDetail(t.Context(), GetJobDetailParams{ID: MockJobID})
	require.NoError(t, err)
	_, ok = dres.(*GetJobDetailFound)
	require.True(t, ok, "expected *GetJobDetailFound, got %T", dres)
}

func TestGetJobDetail(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	res, err := newTestClient(t, srv.URL).GetJobDetail(t.Context(), GetJobDetailParams{ID: MockJobID})
	require.NoError(t, err)

	detail, ok := res.(*DetailResponse)
	require.True(t, ok, "expected *DetailResponse, got %T", res)

	jo := detail.Result.JobOpening
	assert.Equal(t, "https://curtinmaritime.bamboohr.com/careers/201", jo.JobOpeningShareUrl)
	assert.Equal(t, "Vessel Chef", jo.JobOpeningName)
	assert.Equal(t, "Open", jo.JobOpeningStatus)
	assert.Equal(t, NilString{Value: "18376"}, jo.JobCategoryId)
	assert.Equal(t, NilString{Value: "18418"}, jo.DepartmentId)
	assert.Equal(t, NilString{Value: "Marine Transportation"}, jo.DepartmentLabel)
	assert.Equal(t, "Full-Time", jo.EmploymentStatusLabel)
	assert.Equal(t, NilString{Value: "Long Beach"}, jo.Location.City)
	assert.Equal(t, NilString{Value: "California"}, jo.Location.State)
	assert.Equal(t, NilString{Value: "90802"}, jo.Location.PostalCode)
	assert.Equal(t, NilString{Value: "United States"}, jo.Location.AddressCountry)
	assert.Equal(t, NilString{Null: true}, jo.AtsLocation.Country)
	assert.Equal(t, NilString{Null: true}, jo.AtsLocation.CountryId)
	assert.Contains(t, jo.Description, "Curtin Maritime")
	assert.Equal(t, NilString{Value: "$300 - $425/Day ($21.43 - $30.35 per hour)"}, jo.Compensation)
	assert.Equal(t, NilString{Value: "2025-04-22"}, jo.DatePosted)
	assert.Equal(t, NilString{Value: "Experienced"}, jo.MinimumExperience)
	assert.Equal(t, NilString{Value: "0"}, jo.LocationType)
	assert.Equal(t, NilBool{Value: false}, jo.SeekPromoted)
	assert.True(t, detail.Result.FormFields.Set, "formFields must be surfaced")
}

// TestGetJobDetailNulls covers a posting with the optional fields nulled
// out - and the departmentLabel quirk: the detail endpoint reports an empty
// string where the list feed reports null.
func TestGetJobDetailNulls(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	res, err := newTestClient(t, srv.URL).GetJobDetail(t.Context(), GetJobDetailParams{ID: MockNullsJobID})
	require.NoError(t, err)

	detail, ok := res.(*DetailResponse)
	require.True(t, ok, "expected *DetailResponse, got %T", res)

	jo := detail.Result.JobOpening
	assert.Equal(t, "Talent Community @ Giatec", jo.JobOpeningName)
	assert.Equal(t, NilString{Null: true}, jo.JobCategoryId)
	assert.Equal(t, NilString{Null: true}, jo.DepartmentId)
	assert.Equal(t, NilString{Value: ""}, jo.DepartmentLabel)
	assert.Equal(t, NilString{Null: true}, jo.Compensation)
	assert.Equal(t, NilString{Null: true}, jo.MinimumExperience)
	assert.Equal(t, NilString{Value: "2"}, jo.LocationType)
}

func TestGetJobDetailNotFound(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()

	res, err := newTestClient(t, srv.URL).GetJobDetail(t.Context(), GetJobDetailParams{ID: "999999"})
	require.NoError(t, err)

	nf, ok := res.(*NotFoundError)
	require.True(t, ok, "expected *NotFoundError, got %T", res)
	assert.Equal(t, "not_found", nf.Type)
	assert.Equal(t, "Resource not found.", nf.Title)
	assert.Equal(t, OptString{Value: "Looks like the id you provided doesn't exist.", Set: true}, nf.Details)
}
