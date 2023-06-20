output "skysql_endpoint_service_id" {
  description = "SkySQL privatelink endpoint service id"
  value       = data.skysql_service.this.endpoints[0].endpoint_service
}

output "skysql_host" {
  description = "Hostname for private database connections"
  value       = data.skysql_service.this.fqdn
}

output "skysql_cmd" {
  value = "mariadb --host ${data.skysql_service.this.fqdn} --port 3306 --user ${data.skysql_service.this.service_id} -p --ssl-ca ~/skysql_chain_2022.pem"
}
