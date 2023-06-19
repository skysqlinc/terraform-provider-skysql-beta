# MariaDB SkySQL Terraform Provider

> The MariaDB SkySQL Terraform Provider is a Technical Preview. Software in Tech Preview should not be used for production workloads.

This is a Terraform provider for managing resources in [MariaDB SkySQL](https://mariadb.com/products/skysql/).

See the examples in `/docs` in this repository for usage examples.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.18
- Google APIs to be enabled: 
- Cloud Functions API
- Cloud DNS API
- Google APIs to be enabled: Secret manager API
- Serverless VPC Access API
- Cloud Build API

Here's a grouping of permissions that work, 
- Cloud Functions Admin
- Cloud Run Admin
- Compute Instance Admin (v1)
- Compute OS Admin Login
- DNS Administrator
- Kubernetes Engine Admin
- Kubernetes Engine Cluster Admin
- MariaDB Developer Admin
- MariaDB Kubernetes
- MariaDB Login
- MariaDB Storage Admin
- Project IAM Admin
- Secret Manager Admin
- Serverless VPC Access Admin
- Service Account Admin
- Service Account Token Creator
- Service Usage Admin
- Viewer

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```
