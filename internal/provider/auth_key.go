package provider

import (
	"context"
	"time"

	"github.com/getunderpass/terraform-provider-underpass/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// An auth key: machines join the workspace with it. Keys can't be changed,
// so every attribute replaces the key; its secret is only known when it's
// made, and kept (sensitive) in the state.

type authKeyResource struct{ c *client.Client }

func newAuthKeyResource() resource.Resource { return &authKeyResource{} }

type authKeyModel struct {
	ID            types.String `tfsdk:"id"`
	Key           types.String `tfsdk:"key"`
	Prefix        types.String `tfsdk:"prefix"`
	Reusable      types.Bool   `tfsdk:"reusable"`
	Ephemeral     types.Bool   `tfsdk:"ephemeral"`
	PreApproved   types.Bool   `tfsdk:"pre_approved"`
	NoKeyExpiry   types.Bool   `tfsdk:"no_key_expiry"`
	Tags          types.Set    `tfsdk:"tags"`
	ExpirySeconds types.Int64  `tfsdk:"expiry_seconds"`
	ExpiresAt     types.String `tfsdk:"expires_at"`
	Valid         types.Bool   `tfsdk:"valid"`
}

func (r *authKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_key"
}

func (r *authKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceBool := []planmodifier.Bool{boolplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "An auth key, for adding machines to the workspace without signing in on each (servers, containers, CI). " +
			"Keys can't be changed: changing any argument makes a new one. Destroying it revokes it; machines that joined with it stay.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The key itself (`tskey-…`), to pass to `underpass up --auth-key`. Only known when it's made.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"prefix": schema.StringAttribute{
				MarkdownDescription: "The key's first characters, as the dashboard shows them.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"reusable": schema.BoolAttribute{
				MarkdownDescription: "Add any number of machines with it, not just one. Defaults to `false`.",
				Optional:            true, Computed: true, Default: boolDefault(false), PlanModifiers: replaceBool,
			},
			"ephemeral": schema.BoolAttribute{
				MarkdownDescription: "Machines added with it are removed after 10 minutes offline (containers, CI). Defaults to `false`.",
				Optional:            true, Computed: true, Default: boolDefault(false), PlanModifiers: replaceBool,
			},
			"pre_approved": schema.BoolAttribute{
				MarkdownDescription: "Machines added with it skip device approval. Defaults to `false`.",
				Optional:            true, Computed: true, Default: boolDefault(false), PlanModifiers: replaceBool,
			},
			"no_key_expiry": schema.BoolAttribute{
				MarkdownDescription: "Machines added with it never have to sign in again (servers). Defaults to `false`.",
				Optional:            true, Computed: true, Default: boolDefault(false), PlanModifiers: replaceBool,
			},
			"tags": schema.SetAttribute{
				MarkdownDescription: "Tags machines added with it get, like `tag:servers`, for access rules.",
				ElementType:         types.StringType,
				Optional:            true,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.RequiresReplace()},
			},
			"expiry_seconds": schema.Int64Attribute{
				MarkdownDescription: "How long the key works, in seconds: up to 90 days (7776000), the default.",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.Between(60, 7776000)},
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"expires_at": schema.StringAttribute{
				MarkdownDescription: "When the key stops working (RFC 3339).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"valid": schema.BoolAttribute{
				MarkdownDescription: "Whether it can still add a machine (not expired, and unused if it's single-use).",
				Computed:            true,
			},
		},
	}
}

func (r *authKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.c = clientFrom(req.ProviderData)
}

func (r *authKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan authKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tags := []string{}
	resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k, secret, err := r.c.CreateAuthKey(ctx, client.NewAuthKey{
		Reusable:      plan.Reusable.ValueBool(),
		Ephemeral:     plan.Ephemeral.ValueBool(),
		PreApproved:   plan.PreApproved.ValueBool(),
		NoKeyExpiry:   plan.NoKeyExpiry.ValueBool(),
		Tags:          tags,
		ExpirySeconds: plan.ExpirySeconds.ValueInt64(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Couldn't create the auth key", err.Error())
		return
	}
	plan.ID = types.StringValue(k.ID)
	plan.Key = types.StringValue(secret)
	fillAuthKey(&plan, k)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func fillAuthKey(m *authKeyModel, k client.AuthKey) {
	m.Prefix = types.StringValue(k.Prefix)
	m.ExpiresAt = types.StringValue(k.ExpiresAt.UTC().Format(time.RFC3339))
	m.Valid = types.BoolValue(k.Valid)
	m.Reusable = types.BoolValue(k.Reusable)
	m.Ephemeral = types.BoolValue(k.Ephemeral)
	m.PreApproved = types.BoolValue(k.PreApproved)
	m.NoKeyExpiry = types.BoolValue(k.NoKeyExpiry)
	if len(k.Tags) > 0 || !m.Tags.IsNull() {
		vals := make([]attr.Value, 0, len(k.Tags))
		for _, t := range k.Tags {
			vals = append(vals, types.StringValue(t))
		}
		m.Tags = types.SetValueMust(types.StringType, vals)
	}
}

func (r *authKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state authKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k, err := r.c.AuthKey(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		// Revoked in the dashboard, or cleaned up a month after it stopped
		// working: Terraform makes a new one on the next apply.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Couldn't read the auth key", err.Error())
		return
	}
	fillAuthKey(&state, k)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update is never called: every argument replaces the key.
func (r *authKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan authKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *authKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state authKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.c.RevokeAuthKey(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Couldn't revoke the auth key", err.Error())
	}
}
