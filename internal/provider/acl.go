package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/getunderpass/terraform-provider-underpass/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The workspace's access rules, as one JSON document (the format the API and
// docs use: rules, ssh, remote, clipboard, groups, tests).
//
// The API normalizes what it saves (it gives rules IDs and fills in
// defaults), so reading it back never matches the configuration word for
// word. After each apply the provider keeps a fingerprint of what the API
// saved; a later read shows a change only when the rules differ from that,
// that is, when someone changed them outside Terraform.

type aclResource struct{ c *client.Client }

func newACLResource() resource.Resource { return &aclResource{} }

type aclModel struct {
	ID     types.String         `tfsdk:"id"`
	Policy jsontypes.Normalized `tfsdk:"policy"`
}

const savedKey = "saved"

func (r *aclResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl"
}

func (r *aclResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The workspace's [access rules](https://docs.getunderpass.com/features/access-controls), as one JSON document. " +
			"Only one per workspace. Saving is refused when a test in the policy fails. Destroying it goes back to no rules: every machine reaches every other. " +
			"API tokens can't change the remote control and shared clipboard rules; leave them as they are.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"policy": schema.StringAttribute{
				MarkdownDescription: "The policy: `rules`, `ssh`, `remote`, `clipboard`, `groups` and `tests`. Use `jsonencode()`.",
				CustomType:          jsontypes.NormalizedType{},
				Required:            true,
			},
		},
	}
}

func (r *aclResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.c = clientFrom(req.ProviderData)
}

// fingerprint is a short hash of the policy as the API keeps it.
func fingerprint(policy json.RawMessage) string {
	// Through a map, so key order never matters.
	var v any
	if json.Unmarshal(policy, &v) == nil {
		if b, err := json.Marshal(v); err == nil {
			policy = b
		}
	}
	sum := sha256.Sum256(policy)
	return hex.EncodeToString(sum[:])
}

func (r *aclResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan aclModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fp, err := r.save(ctx, plan.Policy.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Couldn't save the access rules", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, savedKey, []byte(`"`+fp+`"`))...)
	plan.ID = types.StringValue("acl")
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// save saves the policy and returns the fingerprint of what the API made of it.
func (r *aclResource) save(ctx context.Context, policy string) (string, error) {
	if err := r.c.SetPolicy(ctx, json.RawMessage(policy)); err != nil {
		return "", err
	}
	saved, _, err := r.c.Policy(ctx)
	if err != nil {
		return "", err
	}
	return fingerprint(saved), nil
}

func (r *aclResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state aclModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	now, custom, err := r.c.Policy(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Couldn't read the access rules", err.Error())
		return
	}
	if !custom {
		// Reset to no rules outside Terraform: it's gone.
		resp.State.RemoveResource(ctx)
		return
	}
	raw, d := req.Private.GetKey(ctx, savedKey)
	resp.Diagnostics.Append(d...)
	var saved string
	_ = json.Unmarshal(raw, &saved)
	if saved != "" && saved == fingerprint(now) {
		return // as Terraform left it: keep the configuration's wording
	}
	// Changed elsewhere (or imported): show what's there.
	state.Policy = jsontypes.NewNormalizedValue(string(now))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, savedKey, []byte(`"`+fingerprint(now)+`"`))...)
}

func (r *aclResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan aclModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fp, err := r.save(ctx, plan.Policy.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Couldn't save the access rules", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, savedKey, []byte(`"`+fp+`"`))...)
	plan.ID = types.StringValue("acl")
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *aclResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	if err := r.c.ResetPolicy(ctx); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Couldn't reset the access rules", err.Error())
	}
}

// ImportState takes the workspace's rules as they are (`terraform import underpass_acl.this acl`).
func (r *aclResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), "acl")...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy"), jsontypes.NewNormalizedValue("{}"))...)
}
