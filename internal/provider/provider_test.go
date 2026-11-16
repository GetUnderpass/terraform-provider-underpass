package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// The provider as Terraform loads it: every resource's and data source's
// schema must be valid, or `terraform init` and every plan would fail.
func TestSchemasAreValid(t *testing.T) {
	server := providerserver.NewProtocol6(New("test")())()
	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		t.Errorf("%s: %s", d.Summary, d.Detail)
	}
	for _, name := range []string{"underpass_auth_key", "underpass_acl", "underpass_settings", "underpass_dns", "underpass_machine"} {
		if resp.ResourceSchemas[name] == nil {
			t.Errorf("no resource %s", name)
		}
	}
	for _, name := range []string{"underpass_machines", "underpass_machine"} {
		if resp.DataSourceSchemas[name] == nil {
			t.Errorf("no data source %s", name)
		}
	}
	if k := resp.ResourceSchemas["underpass_auth_key"]; k != nil {
		for _, a := range k.Block.Attributes {
			if a.Name == "key" && !a.Sensitive {
				t.Error("the auth key's secret isn't marked sensitive")
			}
		}
	}
}

// The access rules' fingerprint ignores key order and spacing, and nothing
// else, so wording changes don't show as drift but real changes do.
func TestFingerprint(t *testing.T) {
	a := fingerprint(json.RawMessage(`{"rules":[{"src":["*"],"dst":["tag:web"]}],"groups":{}}`))
	b := fingerprint(json.RawMessage(`{ "groups": {}, "rules": [ { "dst": ["tag:web"], "src": ["*"] } ] }`))
	c := fingerprint(json.RawMessage(`{"rules":[{"src":["*"],"dst":["tag:db"]}],"groups":{}}`))
	if a != b {
		t.Error("the same policy, ordered differently, has another fingerprint")
	}
	if a == c {
		t.Error("a different policy has the same fingerprint")
	}
}

// What an import reads drops what the API adds (rule IDs, empty sections),
// so a configuration written the usual way plans no change after it.
func TestAsWritten(t *testing.T) {
	got := asWritten(json.RawMessage(`{"rules":[{"id":"7mcv6u7kjw66","src":["*"],"dst":["tag:web"]}],"remote":[],"clipboard":[],"groups":{},"tests":[{"src":"user:a@b.co","accept":["tag:web:22"],"deny":[]}]}`))
	want := `{"rules":[{"dst":["tag:web"],"src":["*"]}],"tests":[{"accept":["tag:web:22"],"src":"user:a@b.co"}]}`
	if got != want {
		t.Errorf("asWritten = %s, want %s", got, want)
	}
	// What people do write stays.
	got = asWritten(json.RawMessage(`{"groups":{"group:eng":["a@b.co"]},"ssh":[{"id":"x","src":["*"]}]}`))
	want = `{"groups":{"group:eng":["a@b.co"]},"ssh":[{"src":["*"]}]}`
	if got != want {
		t.Errorf("asWritten = %s, want %s", got, want)
	}
}
