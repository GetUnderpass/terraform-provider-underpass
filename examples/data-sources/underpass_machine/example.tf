data "underpass_machine" "db" {
  name = "db-eu-1"
}

output "db_address" {
  value = data.underpass_machine.db.ipv4
}
