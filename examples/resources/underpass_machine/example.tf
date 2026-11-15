data "underpass_machine" "router" {
  name = "office-router"
}

resource "underpass_machine" "router" {
  machine_id          = data.underpass_machine.router.id
  name                = "office-router"
  key_expiry_disabled = true
  approved_routes     = ["192.168.1.0/24"]
}
