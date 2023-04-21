variable "project_id" {
  description = "The ID of the project in which the resources will be created."
  type        = string
}

variable "region" {
  description = "The region in which the resources will be created."
  type        = string
}

variable "function_name" {
  description = "The name of the function."
  type        = string
}

variable "source_dir" {
  description = "The directory containing the function code."
  type        = string
}

variable "gcs_bucket" {
  description = "The name of the GCS bucket to upload the function code to."
  type        = string
  default     = ""
}

variable "gcs_bucket_location" {
  description = "The location of the GCS bucket.  Required if gcs_bucket is not set."
  type        = string
  default     = "US"
}

variable "vpc_connector" {
  description = "The name of the VPC connector to use."
  type        = string
  default     = null
}

variable "db_host" {
  description = "The host of the database to connect to."
  type        = string
}

variable "db_user" {
  description = "The user to connect to the database as."
  type        = string
}

variable "db_password_secret" {
  description = "The name of the secret containing the password to connect to the database."
  type        = string
}
