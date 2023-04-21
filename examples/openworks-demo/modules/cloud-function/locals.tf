locals {
  archive_file    = "${var.source_dir}/source.zip"
  gcs_object_name = "${var.function_name}-source.zip"
  bucket_name     = var.gcs_bucket == "" ? "skyquery-${random_id.suffix.hex}" : var.gcs_bucket
  secrets_sa      = "skyquery-sec-${random_id.suffix.hex}"
  invoker_sa      = "skyquery-inv-${random_id.suffix.hex}"
  signed_jwt      = jsondecode(data.http.sign_jwt.response_body)
}
