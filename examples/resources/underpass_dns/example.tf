resource "underpass_dns" "this" {
  name           = "acme"
  https          = true
  search_domains = ["corp.acme.com"]
  split_dns = {
    "corp.acme.com" = ["10.0.0.53"]
  }
}
