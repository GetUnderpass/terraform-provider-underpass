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
