terraform {
  required_version = "~>1.3.7"
  required_providers {
    skysql = {
      source  = "registry.terraform.io/mariadb-corporation/skysql"
      version = "~>1.0.0"
    }
    google = {
      source  = "hashicorp/google"
      version = "~> 4.62"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5.0"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~> 2.3.0"
    }
    http = {
      source  = "hashicorp/http"
      version = "~> 3.2.0"
    }
  }
}

provider "skysql" {}
provider "google" {
  project = var.project_id
  region  = var.region
}
