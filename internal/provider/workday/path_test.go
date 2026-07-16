package workday

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobDetailKeyFromPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		externalPath string
		wantLocation string
		wantTitle    string
		wantOK       bool
	}{
		{
			name:         "typical two-segment path",
			externalPath: "/job/US-CA-Remote/Software-Engineer_JR12345",
			wantLocation: "US-CA-Remote",
			wantTitle:    "Software-Engineer_JR12345",
			wantOK:       true,
		},
		{
			name:         "title containing extra double-dashes still cuts on the first slash",
			externalPath: "/job/US-CA-Remote/Software-Engineer--CUDA_JR12345",
			wantLocation: "US-CA-Remote",
			wantTitle:    "Software-Engineer--CUDA_JR12345",
			wantOK:       true,
		},
		{
			name:         "missing /job/ prefix is rejected",
			externalPath: "US-CA-Remote/Software-Engineer_JR12345",
			wantOK:       false,
		},
		{
			name:         "other leading segment is rejected",
			externalPath: "/details/US-CA-Remote/Software-Engineer_JR12345",
			wantOK:       false,
		},
		{
			name:         "location-less single segment is accepted",
			externalPath: "/job/APSCA-Certified-Social-Compliance-Auditor_JR0019413",
			wantLocation: "",
			wantTitle:    "APSCA-Certified-Social-Compliance-Auditor_JR0019413",
			wantOK:       true,
		},
		{
			name:         "trailing slash (empty titleSlug) fails",
			externalPath: "/job/US-CA-Remote/",
			wantOK:       false,
		},
		{
			name:         "empty location with titleSlug is accepted",
			externalPath: "/job//Software-Engineer_JR12345",
			wantLocation: "",
			wantTitle:    "Software-Engineer_JR12345",
			wantOK:       true,
		},
		{
			name:         "extra path segments fail instead of percent-encoding the slash",
			externalPath: "/job/US/CA/Software-Engineer_JR12345",
			wantOK:       false,
		},
		{
			name:         "bare /job/ with no slug fails",
			externalPath: "/job/",
			wantOK:       false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotLocation, gotTitle, gotOK := JobDetailKeyFromPath(tc.externalPath)
			assert.Equal(t, tc.wantOK, gotOK)
			assert.Equal(t, tc.wantLocation, gotLocation)
			assert.Equal(t, tc.wantTitle, gotTitle)
		})
	}
}

func TestPublicSiteURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		baseURL string
		want    string
		wantErr bool
	}{
		{
			name:    "NVIDIA's tenant shape",
			baseURL: "https://nvidia.wd5.myworkdayjobs.com/wday/cxs/nvidia/NVIDIAExternalCareerSite",
			want:    "https://nvidia.wd5.myworkdayjobs.com/NVIDIAExternalCareerSite",
		},
		{
			name:    "a different tenant/pod/site",
			baseURL: "https://acme.wd3.myworkdayjobs.com/wday/cxs/acme/AcmeCareers",
			want:    "https://acme.wd3.myworkdayjobs.com/AcmeCareers",
		},
		{
			name:    "trailing slash on base URL",
			baseURL: "https://acme.wd3.myworkdayjobs.com/wday/cxs/acme/AcmeCareers/",
			want:    "https://acme.wd3.myworkdayjobs.com/AcmeCareers",
		},
		{
			name:    "no path segments",
			baseURL: "https://acme.wd3.myworkdayjobs.com/",
			wantErr: true,
		},
		{
			name:    "unparseable URL",
			baseURL: "://not-a-url",
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := PublicSiteURL(tc.baseURL)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
