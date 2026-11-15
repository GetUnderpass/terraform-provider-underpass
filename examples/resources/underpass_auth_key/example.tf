resource "underpass_auth_key" "servers" {
  reusable       = true
  pre_approved   = true
  no_key_expiry  = true
  tags           = ["tag:servers"]
  expiry_seconds = 30 * 24 * 3600
}

# On the server: underpass up --auth-key "$KEY"
output "server_key" {
  value     = underpass_auth_key.servers.key
  sensitive = true
}
