package skysql

import (
	"github.com/go-resty/resty/v2"
	"github.com/mariadb-corporation/terraform-provider-skysql-v2/internal/skysql/organization"
)

type Client struct {
	HTTPClient *resty.Client
}

func New(baseURL string, AccessToken string) *Client {
	return &Client{
		HTTPClient: resty.New().
			SetAuthScheme("Bearer").
			SetAuthToken(AccessToken).
			SetBaseURL(baseURL).
			EnableTrace(),
	}
}

func (c *Client) GetProjects() ([]organization.Project, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult([]organization.Project{}).
		Get("/organization/v1/projects")
	if err != nil {
		return nil, err
	}
	return *resp.Result().(*[]organization.Project), err
}
