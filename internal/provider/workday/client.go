package workday

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// TenantClient queries any confirmed Workday tenant by slug.
type TenantClient struct {
	client *Client
}

// NewTenantClient builds a TenantClient with the same options as NewClient.
func NewTenantClient(opts ...ClientOption) (*TenantClient, error) {
	c, err := NewClient("", opts...)
	if err != nil {
		return nil, err
	}
	return &TenantClient{
		client: c,
	}, nil
}

// serverURLByTenant resolves a confirmed tenant to its CXS base URL.
func serverURLByTenant(tenant string) (serverURL *url.URL, err error) {
	company, ok := CompaniesByTenant[strings.ToLower(tenant)]
	if !ok {
		return nil, fmt.Errorf("tenant %s not found", tenant)
	}
	u, err := url.Parse(company.BaseURL())
	if err != nil {
		return nil, fmt.Errorf("parse base URL for tenant %s: %w", tenant, err)
	}
	return u, nil
}

// JobsByTenant searches a confirmed tenant.
func (c *TenantClient) JobsByTenant(ctx context.Context, tenant string, request *JobsRequest) (*JobsResponse, error) {
	serverURL, err := serverURLByTenant(tenant)
	if err != nil {
		return nil, err
	}
	ctx = WithServerURL(ctx, serverURL)
	return c.client.SearchJobs(ctx, request)
}

// ToGetJobDetailParams returns parameters for the first posting with a valid
// ExternalPath, or false when none qualifies.
func (rsp *JobsResponse) ToGetJobDetailParams() (GetJobDetailParams, bool) {
	for _, posting := range rsp.JobPostings {
		externalPath, ok := posting.ExternalPath.Get()
		if !ok {
			continue
		}
		location, titleSlug, ok := JobDetailKeyFromPath(externalPath)
		if !ok {
			continue
		}
		return GetJobDetailParams{Location: location, TitleSlug: titleSlug}, true
	}
	return GetJobDetailParams{}, false
}

// JobDetailByTenant fetches one posting for a confirmed tenant.
func (c *TenantClient) JobDetailByTenant(ctx context.Context, tenant, location, titleSlug string) (*JobDetailResponse, error) {
	serverURL, err := serverURLByTenant(tenant)
	if err != nil {
		return nil, err
	}
	ctx = WithServerURL(ctx, serverURL)
	return c.client.GetJobDetail(ctx, GetJobDetailParams{Location: location, TitleSlug: titleSlug})
}
