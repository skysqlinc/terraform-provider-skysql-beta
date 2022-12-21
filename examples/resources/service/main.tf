terraform {
  required_providers {
    skysql = {
      source = "registry.terraform.io/mariadb-corporation/skysql-v2"
    }
  }
}

provider "skysql" {}


data "skysql_versions" "default" {}


locals {
  sky_versions_filtered = [
    for item in data.skysql_versions.default.versions : item if item.topology == "standalone"
  ]
}


resource "skysql_service" default {
  project_id     = "e95584aa-3d0d-4513-8cbe-5c63d36a2baa"
  service_type   = "transactional"
  topology       = "standalone"
  cloud_provider = "aws"
  region         = "us-east-1"
  name           = "vf-test9"
  architecture   = "amd64"
  nodes          = 1
  size           = "sky-2x8"
  storage        = 100
  ssl_enabled    = true
  version        = local.sky_versions_filtered[0].name
  volume_type    = "gp2"
  wait_for_creation = true
}