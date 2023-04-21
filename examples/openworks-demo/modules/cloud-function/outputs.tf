output "trigger_response" {
  description = "The response from the cloud function trigger"
  value       = data.http.trigger.response_body
}

output "bucket_name" {
  description = "The name of the bucket where the function code is stored"
  value       = local.bucket_name
}

output "gcs_object_name" {
  description = "The name of the object in the bucket where the function code is stored"
  value       = local.gcs_object_name
}

output "inoker_sa_email" {
  description = "The name of the service account used to invoke the function"
  value       = google_service_account.invoker.email
}
