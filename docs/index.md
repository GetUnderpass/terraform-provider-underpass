---
page_title: "Provider: Underpass"
description: |-
  Manage an Underpass workspace as code: auth keys, access rules, settings, DNS and machines.
---

# Underpass Provider

Manage an [Underpass](https://getunderpass.com) workspace as code: auth keys, access rules, settings, DNS and machines.

The provider acts with an **API token**: in the dashboard, open **Keys → API tokens** and create one with **Read & write** scope. A token only ever reaches the workspace it was made in. What API tokens can't change (remote control, shared clipboard, AI agents, terminal and other people's settings) stays in the dashboard; see [the API docs](https://docs.getunderpass.com/manage/api#what-tokens-cant-do).

## Example

```terraform
terraform {
  required_providers {
    underpass = {
      source  = "getunderpass/underpass"
      version = "~> 0.1"
    }
  }
}

# The token from UNDERPASS_API_TOKEN.
provider "underpass" {}

resource "underpass_auth_key" "servers" {
  reusable      = true
  pre_approved  = true
  no_key_expiry = true
  tags          = ["tag:servers"]
}
```

## Schema

### Optional

- `api_token` (String, Sensitive) An API token with **Read & write** scope (`tsapi-…`). Can also be set with `UNDERPASS_API_TOKEN`.
- `api_url` (String) The coordination service. Defaults to `https://coord.getunderpass.com`; set it when you run your own. Can also be set with `UNDERPASS_API_URL`.
