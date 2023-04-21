output "skysql_endpoint_service_id" {
  description = "SkySQL privatelink endpoint service id"
  value       = data.skysql_service.this.endpoints[0].endpoint_service
}

output "skysql_host" {
  description = "Hostname for private database connections"
  value       = var.link_dns ? data.skysql_service.this.fqdn : google_compute_address.this.address
}

output "trigger_response" {
  description = "Response from the cloud function trigger"
  value       = module.cloud_function.trigger_response
}

output "wordpress_url" {
  description = "URL for the wordpress application"
  value       = module.cloud_run.service.status[0].url
}
