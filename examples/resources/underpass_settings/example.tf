resource "underpass_settings" "this" {
  device_approval = true
  key_expiry_days = 90
  auto_update     = true
  lan_games       = true
  lan_games_machines = ["tag:gaming"]
}
