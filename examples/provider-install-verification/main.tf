terraform {
  required_providers {
    skysql = {
      source = "registry.terraform.io/mariadb-corporation/skysql-v2"
    }
  }
}

provider "skysql" {}


data "skysql_projects" "default" {}

output "sky_projects" {
  value = data.skysql_projects.default.projects
}

data "skysql_versions" "default" {}


output "sky_versions" {
  value = data.skysql_versions.default.versions
}


locals {
  sky_versions_filtered = [
    for item in data.skysql_versions.default.versions : item if item.product == "xpand"
]
}


output "sky_versions_xpand" {
  value = local.sky_versions_filtered
}

output "sky_versions_xpand_latest" {
  value = local.sky_versions_filtered[0]
}