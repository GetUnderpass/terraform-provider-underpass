data "underpass_machines" "servers" {
  tag = "tag:servers"
}

output "server_addresses" {
  value = [for m in data.underpass_machines.servers.machines : m.ipv4]
}
