# Terraform provider for Underpass

Manage an [Underpass](https://getunderpass.com) workspace as code: auth keys, access rules, settings, DNS and machines.

```terraform
terraform {
  required_providers {
    underpass = {
      source  = "getunderpass/underpass"
      version = "~> 0.1"
    }
  }
}

provider "underpass" {} # UNDERPASS_API_TOKEN

resource "underpass_auth_key" "servers" {
  reusable      = true
  no_key_expiry = true
  tags          = ["tag:servers"]
}

resource "underpass_acl" "this" {
  policy = jsonencode({
    rules = [{ src = ["group:engineering"], dst = ["tag:servers"], proto = "tcp", ports = ["22", "443"] }]
    groups = { "group:engineering" = ["sara@acme.com"] }
  })
}
```

Make an API token in the dashboard: **Keys → API tokens**, scope **Read & write**. A token only reaches the workspace it was made in.

| Resource | What |
|---|---|
| `underpass_auth_key` | An auth key for adding servers, containers and CI. Its secret is in `key` (sensitive). |
| `underpass_acl` | The access rules, as one JSON document. Saving is refused if a test in it fails. |
| `underpass_settings` | Key expiry, device approval, network logs, automatic updates, LAN games. |
| `underpass_dns` | Nameservers, split DNS, search domains, the workspace's name and HTTPS. |
| `underpass_machine` | A machine already in the workspace: its name, key expiry, approval and approved routes. |

Data sources: `underpass_machines` (all, or by tag) and `underpass_machine` (by name or ID). The full reference is in [docs/](docs/) and on the Terraform Registry.

## Develop

```sh
go test ./...
go run ./tools/gendocs     # docs/ from the schema and examples/
go build -o terraform-provider-underpass
```

To try a local build, point Terraform at it in `~/.terraformrc` (on Windows, `%APPDATA%\terraform.rc`):

```hcl
provider_installation {
  dev_overrides {
    "getunderpass/underpass" = "/path/to/the/folder/with/the/binary"
  }
  direct {}
}
```

## Release

Push a tag like `v0.1.0`. The release workflow builds every platform with GoReleaser and signs the checksums with the release GPG key (the `GPG_PRIVATE_KEY` and `PASSPHRASE` secrets). The Terraform Registry picks up new releases by itself once the provider is published there.

The source lives in the Underpass monorepo under `integrations/terraform-provider`; this repository is its public copy.
