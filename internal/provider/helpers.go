package provider

import (
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func boolDefault(v bool) defaults.Bool { return booldefault.StaticBool(v) }

// boolPtr is a configured bool, or nil when it isn't set.
func boolPtr(b types.Bool) *bool {
	if b.IsNull() || b.IsUnknown() {
		return nil
	}
	v := b.ValueBool()
	return &v
}

// stringList is a list of strings, empty (not null) when there are none.
func stringList(items []string) types.List {
	vals := make([]attr.Value, 0, len(items))
	for _, s := range items {
		vals = append(vals, types.StringValue(s))
	}
	return types.ListValueMust(types.StringType, vals)
}

// stringSet is a set of strings, empty (not null) when there are none.
func stringSet(items []string) types.Set {
	vals := make([]attr.Value, 0, len(items))
	for _, s := range items {
		vals = append(vals, types.StringValue(s))
	}
	return types.SetValueMust(types.StringType, vals)
}

func errFromDiags(d diag.Diagnostics) error {
	var msgs []string
	for _, e := range d.Errors() {
		msgs = append(msgs, e.Summary()+": "+e.Detail())
	}
	return errors.New(strings.Join(msgs, "; "))
}
