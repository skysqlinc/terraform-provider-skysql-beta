terraform {
  required_providers {
    skysql = {
      source = "registry.terraform.io/mariadb-corporation/skysql-v2"
    }
  }
}

provider "skysql" {}


data "skysql_projects" "projects" {}

output "skysql_projects" {
  value = data.skysql_projects.projects
}