---
page_title: "underpass_machines (Data Source)"
description: |-
  The workspace's machines.
---

# underpass_machines (Data Source)

The workspace's machines.

## Example Usage

```terraform
data "underpass_machines" "servers" {
  tag = "tag:servers"
}

output "server_addresses" {
  value = [for m in data.underpass_machines.servers.machines : m.ipv4]
}
```

## Schema

### Optional

- `tag` (String) Only machines with this tag, like `tag:servers`.

### Read-Only

- `machines` (Attributes List) The machines, by name.
