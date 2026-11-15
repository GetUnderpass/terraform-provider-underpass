---
page_title: "underpass_machine (Resource)"
description: |-
  Settings of a machine that's already in the workspace: its name, key expiry, approval and the routes it may serve.
---

# underpass_machine (Resource)

Settings of a machine that's already in the workspace: its name, key expiry, approval and the routes it may serve. Machines join by installing Underpass (an `underpass_auth_key` helps); this doesn't add one. Destroying it stops managing the machine, which stays.

## Example Usage

```terraform
data "underpass_machine" "router" {
  name = "office-router"
}

resource "underpass_machine" "router" {
  machine_id          = data.underpass_machine.router.id
  name                = "office-router"
  key_expiry_disabled = true
  approved_routes     = ["192.168.1.0/24"]
}
```

## Schema

### Required

- `machine_id` (String) The machine's ID (from its page in the dashboard, or the `underpass_machine` data source).

### Optional

- `approved` (Boolean) Approve the machine if it waits for device approval. Approval can't be taken back here; remove the machine in the dashboard instead.
- `approved_routes` (Set of String) Which of the routes the machine offers are approved (subnets like `192.168.1.0/24`, or `0.0.0.0/0` for an exit node).
- `key_expiry_disabled` (Boolean) The machine never has to sign in again (servers).
- `name` (String) The machine's name, which is also its easy name in DNS.

### Read-Only

- `advertised_routes` (Set of String) The routes the machine offers.
- `fqdn` (String) The machine's full name.
- `id` (String) 
- `ipv4` (String) The machine's address in the network.
- `os` (String) Its operating system.
- `tags` (Set of String) Its tags (from the auth key it joined with).
- `user` (String) Who owns it (empty for tagged machines).

## Import

```shell
terraform import underpass_machine.router <machine ID>
```
