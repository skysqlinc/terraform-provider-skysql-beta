package skysql

import (
	"context"
	"errors"
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/mariadb-corporation/terraform-provider-skysql-beta/internal/skysql/organization"
	"github.com/mariadb-corporation/terraform-provider-skysql-beta/internal/skysql/provisioning"
	"net/http"
	"net/url"
	"strconv"
)

type Client struct {
	HTTPClient *resty.Client
}

func New(baseURL string, AccessToken string) *Client {
	transport := logging.NewLoggingHTTPTransport(http.DefaultTransport)

	return &Client{
		HTTPClient: resty.NewWithClient(&http.Client{Transport: transport}).
			SetHeader("User-Agent", "terraform-provider-skysql-beta").
			SetAuthScheme("Bearer").
			SetAuthToken(AccessToken).
			SetBaseURL(baseURL).
			EnableTrace(),
	}
}

func (c *Client) GetProjects(ctx context.Context) ([]organization.Project, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult([]organization.Project{}).
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

func (c *Client) GetServiceByID(ctx context.Context, serviceID string) (*provisioning.Service, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult(provisioning.Service{}).
		SetContext(ctx).
		Get("/provisioning/v1/services/" + serviceID)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}
	return resp.Result().(*provisioning.Service), err
}

func (c *Client) CreateService(ctx context.Context, req *provisioning.CreateServiceRequest) (*provisioning.Service, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult(provisioning.Service{}).
		SetError(ErrorResponse{}).
		SetContext(ctx).
		SetBody(req).
		Post("/provisioning/v1/services")
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, handleError(resp)
	}

	return resp.Result().(*provisioning.Service), err
}

func (c *Client) DeleteServiceByID(ctx context.Context, serviceID string) error {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetError(ErrorResponse{}).
		SetContext(ctx).
		Delete("/provisioning/v1/services/" + serviceID)
	if err != nil {
		return err
	}

	if resp.IsError() {
		return handleError(resp)
	}

	return nil
}

func (c *Client) GetServiceCredentialsByID(ctx context.Context, serviceID string) (*provisioning.Credentials, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult(provisioning.Credentials{}).
		SetContext(ctx).
		Get("/provisioning/v1/services/" + serviceID + "/security/credentials")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}
	return resp.Result().(*provisioning.Credentials), err
}

func (c *Client) UpdateServiceAllowListByID(ctx context.Context, serviceID string, allowlist []provisioning.AllowListItem) ([]provisioning.AllowListItem, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult(provisioning.ReadAllowListResponse{}).
		SetContext(ctx).
		SetBody(allowlist).
		Put("/provisioning/v1/services/" + serviceID + "/security/allowlist")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}
	response := *resp.Result().(*provisioning.ReadAllowListResponse)

	return response[0].AllowList, err
}

func (c *Client) ReadServiceAllowListByID(ctx context.Context, serviceID string) (provisioning.ReadAllowListResponse, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult([]provisioning.AllowListItem{}).
		SetContext(ctx).
		SetResult(provisioning.ReadAllowListResponse{}).
		Get("/provisioning/v1/services/" + serviceID + "/security/allowlist")
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
		errResp := resp.Error().(*ErrorResponse)
		return errors.New(errResp.Errors[0].Message)
	}
	return errors.New(resp.Status())
}
