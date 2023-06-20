###
# VPC Setup
###
resource "google_compute_network" "vpc_network" {
  for_each = toset(var.networks)
  project  = var.project_id
  name     = each.value
}

resource "google_compute_firewall" "allow_iap" {
  for_each = toset(var.networks)
  name     = "${each.value}-allow-iap"
  network  = google_compute_network.vpc_network[each.value].id
  allow {
    protocol = "tcp"
    ports    = ["22"]
  }
  direction     = "INGRESS"
  priority      = 500
  source_ranges = ["35.235.240.0/20"]
  project       = var.project_id
}

resource "google_compute_router" "vpc" {
  for_each = toset(var.networks)
  name     = "${each.value}-router"
  network  = google_compute_network.vpc_network[each.value].id
  region   = var.region
}

resource "google_compute_router_nat" "nat" {
  for_each                           = toset(var.networks)
  name                               = "${each.value}-router-nat"
  router                             = google_compute_router.vpc[each.value].name
  region                             = google_compute_router.vpc[each.value].region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  log_config {
    enable = true
    filter = "ALL"
  }
}

###
# VM Setup
###

resource "google_compute_instance" "vm_instance" {
  for_each     = toset(var.networks)
  name         = "${each.value}-vm"
  zone         = "${var.region}-a"
  machine_type = "e2-medium"

  network_interface {
    network = google_compute_network.vpc_network[each.value].id
  }
  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-11"
      size  = 20
    }
  }
}

###
# Create the SkySQL service
###

data "skysql_versions" "this" {
  topology = var.topology
}

data "google_project" "this" {}

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
  ssl_enabled               = true
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

locals {
  # this should work for all topologies other than lakehouse
  readwrite_port = [for p in data.skysql_service.this.endpoints[0].ports : p.port if p.purpose == "readwrite"][0]
  skysql_domain  = "dev2.skysql.mariadb.net"
}

###
# Creates the private address and forwarding rule for the PSC endpoint
###
resource "google_compute_address" "skysql_psc_address" {
  for_each     = toset(var.networks)
  name         = "${each.value}-${var.skysql_service_name}"
  address_type = "INTERNAL"
  subnetwork   = each.value
  project      = var.project_id
  region       = var.region
}

resource "google_compute_forwarding_rule" "this" {
  for_each              = toset(var.networks)
  name                  = "${each.value}-psc-${var.skysql_service_name}"
  load_balancing_scheme = ""
  region                = var.region
  project               = var.project_id
  ip_address            = google_compute_address.skysql_psc_address[each.value].id
  target                = data.skysql_service.this.endpoints[0].endpoint_service
  network               = google_compute_network.vpc_network[each.value].id
}

###
# Create the private DNS zone and record for skysql dns resolution within the private network
###
resource "google_dns_managed_zone" "skysql_dns" {
  for_each    = toset(var.networks)
  name        = "${each.value}-skysql-psc"
  dns_name    = "${local.skysql_domain}."
  description = "SkySQL PSC forwarding zone"
  visibility  = "private"

  private_visibility_config {
    networks {
      network_url = google_compute_network.vpc_network[each.value].id
    }
  }
}

resource "google_dns_record_set" "skysql_dns" {
  for_each     = toset(var.networks)
  managed_zone = google_dns_managed_zone.skysql_dns[each.value].name
  name         = "${data.skysql_service.this.fqdn}."
  type         = "A"
  ttl          = 300
  rrdatas      = [google_compute_address.skysql_psc_address[each.value].address]
}
