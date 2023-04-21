terraform {
  required_version = "~>1.3.7"
  required_providers {
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
