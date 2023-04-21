data "google_project" "this" {}

data "skysql_versions" "this" {
  topology = var.topology
}

###
# Create the SkySQL service
###
resource "skysql_service" "this" {
  service_type              = "transactional"
  topology                  = var.topology
  cloud_provider            = "gcp"
  region                    = var.region
  name                      = var.skysql_service_name
  architecture              = "amd64"
  nodes                     = 1
  size                      = "sky-2x8"
  storage                   = 100
  ssl_enabled               = false
  version                   = data.skysql_versions.this.versions[0].name
  endpoint_mechanism        = "privateconnect"
  endpoint_allowed_accounts = [data.google_project.this.number]
  wait_for_creation         = true
  # The following line will be required when tearing down the skysql service
  deletion_protection = false
}

data "skysql_service" "this" {
  service_id = skysql_service.this.id
}

data "skysql_credentials" "this" {
  service_id = skysql_service.this.id
}


###
# Creates the private address and forwarding rule for the PSC endpoint
###
resource "google_compute_address" "this" {
  name         = var.skysql_service_name
  address_type = "INTERNAL"
  subnetwork   = var.subnetwork
  project      = var.project_id
  region       = var.region
}

resource "google_compute_forwarding_rule" "this" {
  name                  = "psc-${var.skysql_service_name}"
  load_balancing_scheme = ""
  region                = var.region
  project               = var.project_id
  ip_address            = google_compute_address.this.id
  target                = data.skysql_service.this.endpoints[0].endpoint_service
  network               = var.network
}


###
# Create the private DNS zone and record for skysql dns resolution within the private network
###
data "google_compute_network" "this" {
  name = var.network
}

resource "google_dns_managed_zone" "this" {
  count       = var.link_dns ? 1 : 0
  name        = "skysql-psc"
  dns_name    = "${local.skysql_domain}."
  description = "SkySQL PSC forwarding zone"
  visibility  = "private"

  private_visibility_config {
    networks {
      network_url = data.google_compute_network.this.id
    }
  }
}

resource "google_dns_record_set" "this" {
  count        = var.link_dns ? 1 : 0
  managed_zone = google_dns_managed_zone.this[0].name
  name         = "${data.skysql_service.this.fqdn}."
  type         = "A"
  ttl          = 300
  rrdatas      = [google_compute_address.this.address]
}

###
# Store the skysql credentials in secret manager
###
resource "google_secret_manager_secret" "this" {
  secret_id = local.secret_name
  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_version" "this" {
  secret      = google_secret_manager_secret.this.id
  secret_data = data.skysql_credentials.this.password
}

###
# Create a service account that our application will use to read the secret
###
resource "google_service_account" "secrets_access" {
  account_id   = local.secrets_sa
  display_name = "SkySQL Query Secrets Access"
}

resource "google_project_iam_member" "secrets_access" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.secrets_access.email}"
}


###
# Install wordpress on cloud run
###
module "cloud_run" {
  source     = "github.com/GoogleCloudPlatform/cloud-foundation-fabric//modules/cloud-run?ref=v20.0.0"
  project_id = var.project_id
  name       = var.application_name
  region     = var.region

  containers = [{
    image = "mirror.gcr.io/library/wordpress"
    ports = [{
      name           = "http1"
      protocol       = null
      container_port = 80
    }]
    options = {
      command = null
      args    = null
      env_from = {
        "WORDPRESS_DB_PASSWORD" : {
          key  = "latest"
          name = local.secret_name
        }
      }
      # set up the database connection
      env = {
        "WORDPRESS_DB_HOST" : data.skysql_service.this.fqdn
        "WORDPRESS_DB_NAME" : "wordpress"
        "WORDPRESS_DB_USER" : data.skysql_credentials.this.username
      }
    }
    resources     = null
    volume_mounts = null
  }]

  vpc_connector_create = {
    name          = local.vpc_connector_name
    ip_cidr_range = "10.10.10.0/28"
    vpc_self_link = data.google_compute_network.this.id
  }
  service_account = google_service_account.secrets_access.email
  depends_on      = [google_secret_manager_secret_version.this]
}

###
# Connect to skysql via a cloud function to create the wordpress database
###
module "cloud_function" {
  source             = "./modules/cloud-function"
  project_id         = var.project_id
  region             = var.region
  function_name      = "wordpress-init"
  source_dir         = "${path.module}/init_function"
  vpc_connector      = module.cloud_run.vpc_connector
  db_host            = data.skysql_service.this.fqdn
  db_user            = data.skysql_credentials.this.username
  db_password_secret = local.secret_name
  service_account    = google_service_account.secrets_access.email

  # this is due to a quirk in the cloud function resource that jumbles up the secret name when referenced
  depends_on = [google_secret_manager_secret_version.this]
}
