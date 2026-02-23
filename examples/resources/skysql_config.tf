# Example 1: Create a new service with a custom configuration
#
# Create a custom configuration object with server variable overrides.
# The topology and version must match the target service.
resource "skysql_config" "tuned" {
  name     = "my-tuned-config"
  topology = "es-replica"
  version  = "10.6.7-3-1"

  values = {
    "max_connections"         = "500"
    "innodb_buffer_pool_size" = "2G"
  }
}

# Apply the configuration to a new service at creation time.
# Requires wait_for_creation = true so the service is ready before the config is applied.
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

# Example 2: Apply a custom configuration to an existing service
#
# Step 1: Create a config matching the service's topology and version.
#
# Step 2: Add config_id to the service resource block.
#
# Step 3: Run terraform plan / apply.
#   Terraform will detect that config_id changed from empty to the new config ID.
#   If the service already has this config applied, it will be treated as a no-op.

resource "skysql_config" "custom" {
  name     = "custom-config"
  topology = "es-single"
  version  = "10.6.11-6-1"

  values = {
    "max_connections" = "1000"
  }
}

resource "skysql_service" "existing" {
  # ... all service attributes...
  config_id = skysql_config.custom.id
}
