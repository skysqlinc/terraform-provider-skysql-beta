package skysql

import (
	"context"
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
