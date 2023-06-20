terraform {
  required_version = "~>1.3.7"
  required_providers {
    skysql = {
      source  = "registry.terraform.io/mariadb-corporation/skysql"
      version = "~>1.2.1"
    }
    google = {
      source  = "hashicorp/google"
      version = "~> 4.22"
    }
  }
}

provider "skysql" {}
provider "google" {
  project = var.project_id
  region  = var.region
}
