# Create a custom configuration object
resource "skysql_config" "tuned" {
  name     = "my-tuned-config"
  topology = "es-replica"
  version  = "10.6.7-3-1"

  values = {
    "max_connections"         = "500"
    "innodb_buffer_pool_size" = "2G"
  }
}

# Apply the configuration to a service using config_id on the service resource
resource "skysql_service" "default" {
  service_type      = "transactional"
  topology          = "es-replica"
  cloud_provider    = "aws"
  region            = "us-east-1"
  name              = "myservice"
  architecture      = "amd64"
  nodes             = 1
  size              = "sky-2x8"
  storage           = 100
  ssl_enabled       = true
  version           = "10.6.7-3-1"
  wait_for_creation = true
  config_id         = skysql_config.tuned.id
}
