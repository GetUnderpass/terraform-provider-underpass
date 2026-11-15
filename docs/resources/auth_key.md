---
page_title: "underpass_auth_key (Resource)"
description: |-
  An auth key, for adding machines to the workspace without signing in on each (servers, containers, CI).
---

# underpass_auth_key (Resource)

An auth key, for adding machines to the workspace without signing in on each (servers, containers, CI). Keys can't be changed: changing any argument makes a new one. Destroying it revokes it; machines that joined with it stay.

## Example Usage

```terraform
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
```

## Schema

### Optional

- `ephemeral` (Boolean) Machines added with it are removed after 10 minutes offline (containers, CI). Defaults to `false`.
- `expiry_seconds` (Number) How long the key works, in seconds: up to 90 days (7776000), the default.
- `no_key_expiry` (Boolean) Machines added with it never have to sign in again (servers). Defaults to `false`.
- `pre_approved` (Boolean) Machines added with it skip device approval. Defaults to `false`.
- `reusable` (Boolean) Add any number of machines with it, not just one. Defaults to `false`.
- `tags` (Set of String) Tags machines added with it get, like `tag:servers`, for access rules.

### Read-Only

- `expires_at` (String) When the key stops working (RFC 3339).
- `id` (String) 
- `key` (String, Sensitive) The key itself (`tskey-…`), to pass to `underpass up --auth-key`. Only known when it's made.
- `prefix` (String) The key's first characters, as the dashboard shows them.
- `valid` (Boolean) Whether it can still add a machine (not expired, and unused if it's single-use).
