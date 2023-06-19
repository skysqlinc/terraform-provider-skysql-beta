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

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | ~>1.3.7 |
| <a name="requirement_archive"></a> [archive](#requirement\_archive) | ~> 2.3.0 |
| <a name="requirement_google"></a> [google](#requirement\_google) | ~> 4.62 |
| <a name="requirement_http"></a> [http](#requirement\_http) | ~> 3.2.0 |
| <a name="requirement_random"></a> [random](#requirement\_random) | ~> 3.5.0 |
| <a name="requirement_skysql"></a> [skysql](#requirement\_skysql) | ~>1.0.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_google"></a> [google](#provider\_google) | 4.62.0 |
| <a name="provider_skysql"></a> [skysql](#provider\_skysql) | 1.0.0 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_cloud_function"></a> [cloud\_function](#module\_cloud\_function) | ./modules/cloud-function | n/a |
| <a name="module_cloud_run"></a> [cloud\_run](#module\_cloud\_run) | github.com/GoogleCloudPlatform/cloud-foundation-fabric//modules/cloud-run | v20.0.0 |

## Resources

| Name | Type |
|------|------|
| [google_compute_address.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_address) | resource |
| [google_compute_forwarding_rule.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_forwarding_rule) | resource |
| [google_dns_managed_zone.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_managed_zone) | resource |
| [google_dns_record_set.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_record_set) | resource |
| [google_project_iam_member.secrets_access](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_secret_manager_secret.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/secret_manager_secret) | resource |
| [google_secret_manager_secret_version.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/secret_manager_secret_version) | resource |
| [google_service_account.secrets_access](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| skysql_service.this | resource |
| [google_compute_network.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/compute_network) | data source |
| [google_project.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/project) | data source |
| skysql_credentials.this | data source |
| skysql_service.this | data source |
| skysql_versions.this | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_application_name"></a> [application\_name](#input\_application\_name) | Name of the Cloud Run application to be deployed | `string` | `"openworks-wordpress-demo"` | no |
| <a name="input_link_dns"></a> [link\_dns](#input\_link\_dns) | Flag to enable private dns resolution of the skysql domain name | `bool` | `true` | no |
| <a name="input_network"></a> [network](#input\_network) | VPC network to connect to skysql service | `string` | `"default"` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | GCP project id | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | GCP region | `string` | `"us-central1"` | no |
| <a name="input_skysql_service_name"></a> [skysql\_service\_name](#input\_skysql\_service\_name) | Name of the skysql service being created | `string` | n/a | yes |
| <a name="input_subnetwork"></a> [subnetwork](#input\_subnetwork) | VPC subnetwork to connect to skysql service | `string` | `"default"` | no |
| <a name="input_topology"></a> [topology](#input\_topology) | SkySQL topology type to deploy | `string` | `"es-single"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_skysql_endpoint_service_id"></a> [skysql\_endpoint\_service\_id](#output\_skysql\_endpoint\_service\_id) | SkySQL privatelink endpoint service id |
| <a name="output_skysql_host"></a> [skysql\_host](#output\_skysql\_host) | Hostname for private database connections |
| <a name="output_trigger_response"></a> [trigger\_response](#output\_trigger\_response) | Response from the cloud function trigger |
| <a name="output_wordpress_url"></a> [wordpress\_url](#output\_wordpress\_url) | URL for the wordpress application |
<!-- END_TF_DOCS -->
