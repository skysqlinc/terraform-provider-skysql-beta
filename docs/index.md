

---
page_title: "Provider: MariaDB SkySQL DBaaS API Terraform Provider"
description: |-
The MariaDB SkySQL DBaaS API Terraform Provider allows database services in MariaDB SkySQL to be managed using Terraform.---

# SKYSQL-BETA Provider

The MariaDB SkySQL DBaaS API Terraform Provider allows database services in MariaDB SkySQL to be managed using Terraform:

* It allows SkySQL services to be configured using Terraform's declarative language

* It automatically provisions new SkySQL services when the Terraform configuration is applied

* It automatically tears down SkySQL services when the Terraform configuration is destroyed

[Terraform](https://www.terraform.io/) is an open source infrastructure-as-code (IaC) utility.

The MariaDB SkySQL DBaaS API Terraform Provider is a Technical Preview. Software in Tech Preview should not be used for production workloads.

Alternatively, SkySQL services can be managed interactively using your web browser and the [SkySQL Portal](https://skysql.mariadb.com/dashboard).

Use the navigation to the left to read about the available resources.

## Configure the terraform provider

1. Go to MariaDB ID: https://id.mariadb.com/account/api/ and generate an API key
2. Set environment variables:
```bash
    export TF_SKYSQL_API_ACCESS_TOKEN=[SKYSQL API access token]
    export TF_SKYSQL_API_BASE_URL=https://api.mariadb.com
```

## Example Usage

```terraform
terraform {
  required_providers {
    skysql = {
      source = "registry.terraform.io/mariadb-corporation/skysql-beta"
    }
  }
}

provider "skysql" {}

# Retrieve the list of available versions for each topology like standalone, masterslave, xpand-direct etc
data "skysql_versions" "default" {}


# Filter the list of versions to only include  versions for the standalone topology
locals {
  sky_versions_filtered = [
    for item in data.skysql_versions.default.versions : item if item.topology == "standalone"
  ]
}

# Retrieve the list of projects. Project is a way of grouping the services.
# Note: Next release will make project_id optional in the create service api
data "skysql_projects" "default" {}

output "skysql_projects" {
  value = data.skysql_projects.default
}

# Create a service
resource "skysql_service" "default" {
  project_id     = data.skysql_projects.default.projects[0].id
  service_type   = "transactional"
  topology       = "standalone"
  cloud_provider = "aws"
  region         = "us-east-1"
  name           = "vf-test"
  architecture   = "amd64"
  nodes          = 1
  size           = "sky-2x8"
  storage        = 100
  ssl_enabled    = true
  version        = local.sky_versions_filtered[0].name
  volume_type    = "gp2"
  # The service create is an asynchronous operation.
  # if you want to wait for the service to be created set wait_for_creation to true
  wait_for_creation = true
}

# Retrieve the service default credentials.
# When the service is created please change the default credentials
data "skysql_credentials" "default" {
  service_id = skysql_service.default.id
}

# Retrieve the service details
data "skysql_service" "default" {
  service_id = skysql_service.default.id
}

# Show the service details
output "skysql_service" {
  value = data.skysql_service.default
}

# Show the service credentials
output "skysql_credentials" {
  value     = data.skysql_credentials.default
  sensitive = true
}

# You need to add your ip address in the CIRD format to allow list in order to connect to the service
# Note: the operation is asynchronous by default.
# If you want to wait for the operation to complete set wait_for_creation to true
resource "skysql_allow_list" "default" {
  service_id = skysql_service.default.id
  allow_list = [
    {
      "ip" : "104.28.203.45/32",
      "comment" : "homeoffice"
    }
  ]
  wait_for_creation = true
}

# Example how you can generate a command line for the database connection
output "skysql_cmd" {
  value = "mariadb --host ${data.skysql_service.default.fqdn} --port 3306 --user ${data.skysql_service.default.service_id} -p --ssl-ca ~/Downloads/skysql_chain_2022.pem"
}
```

## Limitations

* The terraform resource `skysql_service` doesn't support updates. If you need to change the configuration of a service, you need to destroy the service and create a new one.

### Secrets and Terraform state

Some resources that can be created with this provider, like `skysql_credentials`, are
considered "secrets", and as such are marked by this provider as _sensitive_, so to
help practitioner to not accidentally leak their value in logs or other form of output.

It's important to remember that the values that constitute the "state" of those
resources will be stored in the [Terraform state](https://www.terraform.io/language/state) file.
This includes the "secrets", that will be part of the state file *unencrypted*.

Because of these limitations, **use of these resources for production deployments is _not_ recommended**.
Failing that, **protecting the content of the state file is strongly recommended**.

The more general advice is that it's better to generate "secrets" outside of Terraform,
and then distribute them securely to the system where Terraform will make use of them.
