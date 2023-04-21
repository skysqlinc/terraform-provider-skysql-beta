<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | ~>1.3.7 |
| <a name="requirement_archive"></a> [archive](#requirement\_archive) | ~> 2.3.0 |
| <a name="requirement_google"></a> [google](#requirement\_google) | ~> 4.62 |
| <a name="requirement_http"></a> [http](#requirement\_http) | ~> 3.2.0 |
| <a name="requirement_random"></a> [random](#requirement\_random) | ~> 3.5.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_archive"></a> [archive](#provider\_archive) | ~> 2.3.0 |
| <a name="provider_google"></a> [google](#provider\_google) | ~> 4.62 |
| <a name="provider_http"></a> [http](#provider\_http) | ~> 3.2.0 |
| <a name="provider_random"></a> [random](#provider\_random) | ~> 3.5.0 |
| <a name="provider_time"></a> [time](#provider\_time) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [google_cloudfunctions_function.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloudfunctions_function) | resource |
| [google_cloudfunctions_function_iam_member.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloudfunctions_function_iam_member) | resource |
| [google_service_account.invoker](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_storage_bucket.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket) | resource |
| [google_storage_bucket_object.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket_object) | resource |
| [random_id.suffix](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/id) | resource |
| [time_sleep.wait_for_iam](https://registry.terraform.io/providers/hashicorp/time/latest/docs/resources/sleep) | resource |
| [time_static.timestamp](https://registry.terraform.io/providers/hashicorp/time/latest/docs/resources/static) | resource |
| [archive_file.this](https://registry.terraform.io/providers/hashicorp/archive/latest/docs/data-sources/file) | data source |
| [google_project.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/project) | data source |
| [google_service_account_jwt.invoker](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/service_account_jwt) | data source |
| [http_http.sign_jwt](https://registry.terraform.io/providers/hashicorp/http/latest/docs/data-sources/http) | data source |
| [http_http.trigger](https://registry.terraform.io/providers/hashicorp/http/latest/docs/data-sources/http) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_db_host"></a> [db\_host](#input\_db\_host) | The host of the database to connect to. | `string` | n/a | yes |
| <a name="input_db_password_secret"></a> [db\_password\_secret](#input\_db\_password\_secret) | The name of the secret containing the password to connect to the database. | `string` | n/a | yes |
| <a name="input_db_user"></a> [db\_user](#input\_db\_user) | The user to connect to the database as. | `string` | n/a | yes |
| <a name="input_function_name"></a> [function\_name](#input\_function\_name) | The name of the function. | `string` | n/a | yes |
| <a name="input_gcs_bucket"></a> [gcs\_bucket](#input\_gcs\_bucket) | The name of the GCS bucket to upload the function code to. | `string` | `""` | no |
| <a name="input_gcs_bucket_location"></a> [gcs\_bucket\_location](#input\_gcs\_bucket\_location) | The location of the GCS bucket.  Required if gcs\_bucket is not set. | `string` | `"US"` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The ID of the project in which the resources will be created. | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | The region in which the resources will be created. | `string` | n/a | yes |
| <a name="input_service_account"></a> [service\_account](#input\_service\_account) | The service account to run the function as. | `string` | n/a | yes |
| <a name="input_source_dir"></a> [source\_dir](#input\_source\_dir) | The directory containing the function code. | `string` | n/a | yes |
| <a name="input_vpc_connector"></a> [vpc\_connector](#input\_vpc\_connector) | The name of the VPC connector to use. | `string` | `null` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_bucket_name"></a> [bucket\_name](#output\_bucket\_name) | The name of the bucket where the function code is stored |
| <a name="output_gcs_object_name"></a> [gcs\_object\_name](#output\_gcs\_object\_name) | The name of the object in the bucket where the function code is stored |
| <a name="output_inoker_sa_email"></a> [inoker\_sa\_email](#output\_inoker\_sa\_email) | The name of the service account used to invoke the function |
| <a name="output_trigger_response"></a> [trigger\_response](#output\_trigger\_response) | The response from the cloud function trigger |
<!-- END_TF_DOCS -->