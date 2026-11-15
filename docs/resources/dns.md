---
page_title: "underpass_dns (Resource)"
description: |-
  The workspace's [DNS settings](https://docs.getunderpass.com/features/easy-names).
---

# underpass_dns (Resource)

The workspace's [DNS settings](https://docs.getunderpass.com/features/easy-names). One per workspace, and it owns all of them: what isn't set here is cleared. Destroying it clears them all (machines keep their own DNS, names stay on the shared domain).

## Example Usage

```terraform
resource "underpass_dns" "this" {
  name           = "acme"
  https          = true
  search_domains = ["corp.acme.com"]
  split_dns = {
    "corp.acme.com" = ["10.0.0.53"]
  }
}
```

## Schema

### Optional

- `https` (Boolean) Let machines get HTTPS certificates for their names.
- `name` (String) The workspace's own name, making its domain `<name>.net.getunderpass.com`. Not set: the shared domain.
- `nameservers` (List of String) Nameservers that get every DNS query from the workspace's machines. Empty: machines keep their own DNS for other names.
- `search_domains` (List of String) Domains tried for short names, after the workspace's own.
- `split_dns` (Map of List of String) Names under a domain go to that domain's own nameservers: `{ "corp.example.com" = ["10.0.0.53"] }`.

### Read-Only

- `domain` (String) The workspace's domain, as machines are named under it.
- `id` (String) 
