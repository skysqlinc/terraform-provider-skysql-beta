
variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
}

variable "networks" {
  description = "List of GCP networks to create"
  type        = list(string)
}

variable "topology" {
  description = "SkySQL topology"
  type        = string
  default     = "standalone"
}

variable "skysql_service_name" {
  description = "SkySQL service name"
  type        = string
}
