terraform {
  required_providers {
    underpass = {
      source  = "getunderpass/underpass"
      version = "~> 0.1"
    }
  }
}

# The API token comes from UNDERPASS_API_TOKEN (Keys → API tokens, Read & write).
provider "underpass" {}

# Servers join with this key and never have to sign in again.
resource "underpass_auth_key" "servers" {
  reusable       = true
  pre_approved   = true
  no_key_expiry  = true
  tags           = ["tag:servers"]
  expiry_seconds = 30 * 24 * 3600
}

# Engineering reaches the servers on SSH and HTTPS; everyone reaches the printers.
resource "underpass_acl" "this" {
  policy = jsonencode({
    groups = {
      "group:engineering" = ["sara@acme.com", "tom@acme.com"]
    }
    rules = [
      { src = ["group:engineering"], dst = ["tag:servers"], proto = "tcp", ports = ["22", "443"] },
      { src = ["*"], dst = ["tag:printers"], proto = "any", ports = ["*"] },
    ]
    tests = [
      { src = "tom@acme.com", accept = ["tag:servers:22"] },
    ]
  })
}

resource "underpass_settings" "this" {
  device_approval = true
  key_expiry_days = 90
  auto_update     = true
}

resource "underpass_dns" "this" {
  name           = "acme"
  https          = true
  search_domains = ["corp.acme.com"]
  split_dns = {
    "corp.acme.com" = ["10.0.0.53"]
  }
}

# A machine that joined with the key: name it, and approve the subnet it offers.
data "underpass_machine" "router" {
  name = "office-router"
}

resource "underpass_machine" "router" {
  machine_id      = data.underpass_machine.router.id
  name            = "office-router"
  approved_routes = ["192.168.1.0/24"]
}

output "server_key" {
  value     = underpass_auth_key.servers.key
  sensitive = true
}
