---
page_title: "underpass_acl (Resource)"
description: |-
  The workspace's [access rules](https://docs.getunderpass.com/features/access-controls), as one JSON document.
---

# underpass_acl (Resource)

The workspace's [access rules](https://docs.getunderpass.com/features/access-controls), as one JSON document. Only one per workspace. Saving is refused when a test in the policy fails. Destroying it goes back to no rules: every machine reaches every other. API tokens can't change the remote control and shared clipboard rules; leave them as they are.

## Example Usage

```terraform
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
      { src = "user:tom@acme.com", accept = ["tag:servers:22"] },
    ]
  })
}
```

## Schema

### Required

- `policy` (String) The policy: `rules`, `ssh`, `remote`, `clipboard`, `groups` and `tests`. Use `jsonencode()`.

### Read-Only

- `id` (String) 

## Import

```shell
terraform import underpass_acl.this acl
```
