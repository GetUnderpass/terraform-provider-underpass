---
page_title: "underpass_settings (Resource)"
description: |-
  Workspace settings.
---

# underpass_settings (Resource)

Workspace settings. Only the ones you set are managed; the rest stay as they are in the dashboard. One per workspace. Destroying it changes nothing: the settings stay as they were last applied. Remote control, shared clipboard, file drops, wake, restart, AI agents and terminal settings are changed by people in the dashboard, not API tokens.

## Example Usage

```terraform
resource "underpass_settings" "this" {
  device_approval = true
  key_expiry_days = 90
  auto_update     = true
  lan_games       = true
  lan_games_machines = ["tag:gaming"]
}
```

## Schema

### Optional

- `auto_update` (Boolean) Machines update themselves as soon as a new version is out.
- `device_approval` (Boolean) New machines wait for an owner or admin to approve them.
- `key_expiry_days` (Number) How many days machines stay signed in before signing in again: 1 to 365.
- `key_expiry_off` (Boolean) Machines never have to sign in again.
- `lan_games` (Boolean) Carry machines' local-network broadcasts to each other, so LAN games find each other.
- `lan_games_machines` (List of String) Which machines LAN games covers, as selectors (`*`, `tag:gaming`, `machine:<id>`, `user:<email>`). Empty: all of them.
- `network_logs` (Boolean) Keep logs of who connects to what (machines report their connections).

### Read-Only

- `id` (String) 
