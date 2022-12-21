terraform {
  required_providers {
    skysql = {
      source = "registry.terraform.io/mariadb-corporation/skysql-v2"
    }
  }
}

provider "skysql" {}

data "skysql_service" "default" {
  id = "dbdwf15185084"
}

output "skysql_service" {
  value = data.skysql_service.default
}
