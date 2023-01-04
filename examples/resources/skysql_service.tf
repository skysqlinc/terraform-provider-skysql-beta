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