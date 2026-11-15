---
page_title: "underpass_machine (Data Source)"
description: |-
  One machine, by name or ID.
---

# underpass_machine (Data Source)

One machine, by name or ID.

## Example Usage

```terraform
data "underpass_machine" "db" {
  name = "db-eu-1"
}

output "db_address" {
  value = data.underpass_machine.db.ipv4
}
```

## Schema

### Optional

- `id` (String) The machine's ID. Give this or `name`.
- `name` (String) The machine's name. Give this or `id`.

### Read-Only

- `advertised_routes` (Set of String) The routes it offers.
- `approved_routes` (Set of String) The routes approved for it.
- `client_version` (String) The Underpass version it runs.
- `fqdn` (String) Its full name.
- `ipv4` (String) Its address in the network.
- `last_seen` (String) When it was last connected (RFC 3339).
- `online` (Boolean) Whether it's connected now.
- `os` (String) Its operating system.
- `tags` (Set of String) Its tags.
- `user` (String) Who owns it (empty for tagged machines).
