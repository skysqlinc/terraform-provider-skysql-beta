package skysql

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"

	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql/autonomous"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql/organization"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql/provisioning"
)

type Client struct {
	HTTPClient *resty.Client
}

// RequestOption is a function that modifies a resty request
type RequestOption func(*resty.Request)

// WithOrgID returns a RequestOption that sets the X-MDB-Org header
// This allows requests to operate in the context of a specific organization
func WithOrgID(orgID string) RequestOption {
	return func(r *resty.Request) {
		if orgID != "" {
			r.SetHeader("X-MDB-Org", orgID)
		}
	}
}

func New(baseURL string, apiKey string) *Client {
	transport := logging.NewLoggingHTTPTransport(http.DefaultTransport)

	clientName, _ := os.Executable()

	return &Client{
		HTTPClient: resty.NewWithClient(&http.Client{Transport: transport}).
			SetHeader("User-Agent", filepath.Base(clientName)).
			SetHeader("X-API-Key", apiKey).
			SetBaseURL(baseURL).
			// Set retry count too non-zero to enable retries
			SetRetryCount(3).
			// Default is 100 milliseconds.
			SetRetryWaitTime(5 * time.Second).
			// MaxWaitTime can be overridden as well.
			// Default is 2 seconds.
			SetRetryMaxWaitTime(20 * time.Second).
			// SetRetryAfter sets callback to calculate wait time between retries.
			// Default (nil) implies exponential backoff with jitter
			SetRetryAfter(func(client *resty.Client, resp *resty.Response) (time.Duration, error) {
				return 0, errors.New("retries quota exceeded")
			}).
			AddRetryCondition(
				// RetryConditionFunc type is for retry condition function
				// input: non-nil Response OR request execution error
				func(r *resty.Response, err error) bool {
					return r.StatusCode() == http.StatusInternalServerError
				}).
			EnableTrace(),
	}
}

func (c *Client) GetProjects(ctx context.Context) ([]organization.Project, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult([]organization.Project{}).
		SetError(&ErrorResponse{}).
		SetContext(ctx).
		Get("/organization/v1/projects")
	if resp.IsError() {
		return nil, handleError(resp)
	}
	return *resp.Result().(*[]organization.Project), err
}

func WithPageSize(value uint) func(url.Values) {
	return func(values url.Values) {
		values.Set("page_size", strconv.Itoa(int(value)))
	}
}

func (c *Client) GetVersions(ctx context.Context, options ...func(url.Values)) ([]provisioning.Version, error) {
	request := c.HTTPClient.R()
	for _, option := range options {
		option(request.QueryParam)
	}
	resp, err := request.
		SetHeader("Accept", "application/json").
		SetResult([]provisioning.Version{}).
		SetError(&ErrorResponse{}).
		SetContext(ctx).
		Get("/provisioning/v1/versions")
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, handleError(resp)
	}
	return *resp.Result().(*[]provisioning.Version), err
}

func (c *Client) GetServiceByID(ctx context.Context, serviceID string, opts ...RequestOption) (*provisioning.Service, error) {
	req := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult(provisioning.Service{}).
		SetError(&ErrorResponse{}).
		SetContext(ctx)

	// Apply request options
	for _, opt := range opts {
		opt(req)
	}

	resp, err := req.Get("/provisioning/v1/services/" + serviceID)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}
	return resp.Result().(*provisioning.Service), err
}

func (c *Client) CreateService(ctx context.Context, req *provisioning.CreateServiceRequest, opts ...RequestOption) (*provisioning.Service, error) {
	r := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult(provisioning.Service{}).
		SetError(&ErrorResponse{}).
		SetContext(ctx).
		SetBody(req)

	// Apply request options
	for _, opt := range opts {
		opt(r)
	}

	resp, err := r.Post("/provisioning/v1/services")
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, handleError(resp)
	}

	return resp.Result().(*provisioning.Service), err
}

func (c *Client) DeleteServiceByID(ctx context.Context, serviceID string, opts ...RequestOption) error {
	req := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetError(&ErrorResponse{}).
		SetContext(ctx)

	// Apply request options
	for _, opt := range opts {
		opt(req)
	}

	resp, err := req.Delete("/provisioning/v1/services/" + serviceID)
	if err != nil {
		return err
	}

	if resp.IsError() {
		return handleError(resp)
	}

	return nil
}

func (c *Client) GetServiceCredentialsByID(ctx context.Context, serviceID string, opts ...RequestOption) (*provisioning.Credentials, error) {
	req := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult(provisioning.Credentials{}).
		SetError(&ErrorResponse{}).
		SetContext(ctx)

	// Apply request options
	for _, opt := range opts {
		opt(req)
	}

	resp, err := req.Get("/provisioning/v1/services/" + serviceID + "/security/credentials")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}
	return resp.Result().(*provisioning.Credentials), err
}

func (c *Client) UpdateServiceAllowListByID(ctx context.Context, serviceID string, allowlist []provisioning.AllowListItem, opts ...RequestOption) ([]provisioning.AllowListItem, error) {
	req := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult(provisioning.ReadAllowListResponse{}).
		SetError(&ErrorResponse{}).
		SetContext(ctx).
		SetBody(allowlist)

	// Apply request options
	for _, opt := range opts {
		opt(req)
	}

	resp, err := req.Put("/provisioning/v1/services/" + serviceID + "/security/allowlist")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}
	response := *resp.Result().(*provisioning.ReadAllowListResponse)

	return response[0].AllowList, err
}

func (c *Client) ReadServiceAllowListByID(ctx context.Context, serviceID string, opts ...RequestOption) (provisioning.ReadAllowListResponse, error) {
	req := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetResult(provisioning.ReadAllowListResponse{}).
		SetError(&ErrorResponse{})

	// Apply request options
	for _, opt := range opts {
		opt(req)
	}

	resp, err := req.Get("/provisioning/v1/services/" + serviceID + "/security/allowlist")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}
	response := *resp.Result().(*provisioning.ReadAllowListResponse)
	if response == nil {
		response = make(provisioning.ReadAllowListResponse, 0)
	}
	return response, err
}

func handleError(resp *resty.Response) error {
	if resp.StatusCode() == 404 {
		return ErrorServiceNotFound
	}
	if resp.StatusCode() == 401 {
		return ErrorUnauthorized
	}
	if resp.Error() != nil {
		if resp.StatusCode() == 500 {
			return errors.New("SkySQL API returned 500 Internal Server Error")
		}
		errResp := resp.Error().(*ErrorResponse)
		return errors.New(errResp.Errors[0].Message)
	}
	return errors.New(resp.Status())
}

func (c *Client) SetServicePowerState(ctx context.Context, serviceID string, isActive bool, opts ...RequestOption) error {
	req := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetBody(&provisioning.PowerStateRequest{IsActive: isActive}).
		SetError(&ErrorResponse{})

	// Apply request options
	for _, opt := range opts {
		opt(req)
	}

	resp, err := req.Post("/provisioning/v1/services/" + serviceID + "/power")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return handleError(resp)
	}

	return err
}

func (c *Client) ModifyServiceEndpoints(
	ctx context.Context,
	serviceID string,
	mechanism string,
	allowedAccounts []string,
	visibility string,
	opts ...RequestOption,
) (*provisioning.ServiceEndpoint, error) {
	req := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetBody(&provisioning.PatchServiceEndpointsRequest{
			{Mechanism: mechanism,
				AllowedAccounts: allowedAccounts,
				Visibility:      visibility},
		}).
		SetResult(provisioning.PatchServiceEndpointsResponse{}).
		SetError(&ErrorResponse{})

	// Apply request options
	for _, opt := range opts {
		opt(req)
	}

	resp, err := req.Patch("/provisioning/v1/services/" + serviceID + "/endpoints")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}
	response := *resp.Result().(*provisioning.PatchServiceEndpointsResponse)
	if response == nil {
		response = make(provisioning.PatchServiceEndpointsResponse, 0)
	}
	return &response[0], err
}

func (c *Client) ModifyServiceSize(ctx context.Context, serviceID string, size string, opts ...RequestOption) error {
	req := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetBody(&provisioning.UpdateServiceSizeRequest{Size: size}).
		SetError(&ErrorResponse{})

	// Apply request options
	for _, opt := range opts {
		opt(req)
	}

	resp, err := req.Post("/provisioning/v1/services/" + serviceID + "/size")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return handleError(resp)
	}

	return err
}

func (c *Client) ModifyServiceNodeNumber(ctx context.Context, serviceID string, nodes int64, opts ...RequestOption) error {
	req := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetBody(&provisioning.UpdateServiceNodesNumberRequest{Nodes: nodes}).
		SetError(&ErrorResponse{})

	// Apply request options
	for _, opt := range opts {
		opt(req)
	}

	resp, err := req.Post("/provisioning/v1/services/" + serviceID + "/nodes")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return handleError(resp)
	}

	return err
}

func (c *Client) ModifyServiceStorage(ctx context.Context, serviceID string, size int64, iops int64, throughput int64, opts ...RequestOption) error {
	req := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetBody(&provisioning.UpdateStorageRequest{Size: size, IOPS: iops, Throughput: throughput}).
		SetError(&ErrorResponse{})

	// Apply request options
	for _, opt := range opts {
		opt(req)
	}

	resp, err := req.Patch("/provisioning/v1/services/" + serviceID + "/storage")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return handleError(resp)
	}

	return err
}

func (c *Client) UpdateServiceTags(ctx context.Context, serviceID string, tags map[string]string, opts ...RequestOption) error {
	req := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetBody(&provisioning.UpdateServiceTagsRequest{Tags: tags}).
		SetError(&ErrorResponse{})

	// Apply request options
	for _, opt := range opts {
		opt(req)
	}

	resp, err := req.Patch("/provisioning/v1/services/" + serviceID + "/tags")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return handleError(resp)
	}

	return err
}

func (c *Client) SetAutonomousActions(
	ctx context.Context,
	value autonomous.SetAutonomousActionsRequest,
) ([]autonomous.ActionResponse, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetBody(value).
		SetResult([]autonomous.ActionResponse{}).
		SetError(&ErrorResponse{}).
		Post("/als/v1/actions")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}

	response := *resp.Result().(*[]autonomous.ActionResponse)
	if response == nil {
		response = make([]autonomous.ActionResponse, 0)
	}
	return response, err
}

func (c *Client) GetAutonomousActions(ctx context.Context, serviceID string) ([]autonomous.ActionResponse, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetResult([]autonomous.ActionResponse{}).
		SetError(&ErrorResponse{}).
		SetQueryParam("service_id", serviceID).
		Get("/als/v1/actions")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}

	response := *resp.Result().(*[]autonomous.ActionResponse)
	if response == nil {
		response = make([]autonomous.ActionResponse, 0)
	}
	return response, err
}

func (c *Client) DeleteAutonomousAction(ctx context.Context, actionID string) error {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetError(&ErrorResponse{}).
		Delete("/als/v1/actions/" + actionID)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return handleError(resp)
	}

	return err
}

func (c *Client) GetAvailabilityZones(ctx context.Context, region string, options ...func(url.Values)) ([]provisioning.AvailabilityZone, error) {
	request := c.HTTPClient.R()
	for _, option := range options {
		option(request.QueryParam)
	}
	resp, err := request.
		SetHeader("Accept", "application/json").
		SetResult([]provisioning.AvailabilityZone{}).
		SetError(&ErrorResponse{}).
		SetContext(ctx).
		Get("/provisioning/v1/regions/" + region + "/zones")
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, handleError(resp)
	}
	return *resp.Result().(*[]provisioning.AvailabilityZone), err
}
