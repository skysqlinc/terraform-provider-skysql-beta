locals {
  # this port lookup should work for all topologies other than lakehouse
  readwrite_port     = [for p in data.skysql_service.this.endpoints[0].ports : p.port if p.purpose == "readwrite"][0]
  skysql_domain      = "dev2.skysql.mariadb.net"
  secret_name        = "skysql-credentials"
  vpc_connector_name = "skysql-vpc-connector"
  secrets_sa         = "openworks-secrets-access"
}

