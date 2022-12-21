package skysql

import (
	"context"
	"errors"
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/mariadb-corporation/terraform-provider-skysql-v2/internal/skysql/organization"
	"github.com/mariadb-corporation/terraform-provider-skysql-v2/internal/skysql/provisioning"
	"net/http"
)

type Client struct {
	HTTPClient *resty.Client
}

func New(baseURL string, AccessToken string) *Client {
	transport := logging.NewLoggingHTTPTransport(http.DefaultTransport)

	return &Client{
		HTTPClient: resty.NewWithClient(&http.Client{Transport: transport}).
			SetHeader("User-Agent", "terraform-provider-skysql-v2").
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
	if err != nil {
		return nil, err
	}
	return *resp.Result().(*[]organization.Project), err
}

func (c *Client) GetVersions(ctx context.Context) ([]provisioning.Version, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult([]provisioning.Version{}).
		SetContext(ctx).
		Get("/provisioning/v1/versions")
	if err != nil {
		return nil, err
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
		if resp.StatusCode() == 404 {
			return nil, ErrorServiceNotFound
		}
		errResp := resp.Error().(*ErrorResponse)
		return nil, errors.New(errResp.Errors[0].Message)
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
		errResp := resp.Error().(*ErrorResponse)
		return errors.New(errResp.Errors[0].Message)
	}

	return nil
}
