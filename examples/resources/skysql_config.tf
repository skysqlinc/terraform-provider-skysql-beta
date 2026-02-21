resource "skysql_config" "tuned" {
  name     = "my-tuned-config"
  topology = "es-replica"
  version  = "10.6.7-3-1"

  values = {
    "max_connections"         = "500"
    "innodb_buffer_pool_size" = "2G"
  }
}
