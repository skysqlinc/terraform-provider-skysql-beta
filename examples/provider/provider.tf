terraform {
  required_providers {
    skysql = {
      source = "registry.terraform.io/mariadb-corporation/skysql-v2"
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
