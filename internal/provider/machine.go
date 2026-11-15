package provider

import (
	"context"
	"slices"

	"github.com/getunderpass/terraform-provider-underpass/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// A machine that's already in the workspace (machines join by installing
// Underpass, often with an underpass_auth_key): its name, key expiry,
// approval and routes. Destroying it stops managing it; the machine stays.

type machineResource struct{ c *client.Client }

func newMachineResource() resource.Resource { return &machineResource{} }

type machineModel struct {
	ID                types.String `tfsdk:"id"`
	MachineID         types.String `tfsdk:"machine_id"`
	Name              types.String `tfsdk:"name"`
	KeyExpiryDisabled types.Bool   `tfsdk:"key_expiry_disabled"`
	Approved          types.Bool   `tfsdk:"approved"`
	ApprovedRoutes    types.Set    `tfsdk:"approved_routes"`
	AdvertisedRoutes  types.Set    `tfsdk:"advertised_routes"`
	IPv4              types.String `tfsdk:"ipv4"`
	FQDN              types.String `tfsdk:"fqdn"`
	OS                types.String `tfsdk:"os"`
	Tags              types.Set    `tfsdk:"tags"`
	User              types.String `tfsdk:"user"`
}

func (r *machineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine"
}

func (r *machineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	computedString := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: desc, Computed: true}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Settings of a machine that's already in the workspace: its name, key expiry, approval and the routes it may serve. " +
			"Machines join by installing Underpass (an `underpass_auth_key` helps); this doesn't add one. Destroying it stops managing the machine, which stays.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"machine_id": schema.StringAttribute{
				MarkdownDescription: "The machine's ID (from its page in the dashboard, or the `underpass_machine` data source).",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The machine's name, which is also its easy name in DNS.",
				Optional:            true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"key_expiry_disabled": schema.BoolAttribute{
				MarkdownDescription: "The machine never has to sign in again (servers).",
				Optional:            true,
			},
			"approved": schema.BoolAttribute{
				MarkdownDescription: "Approve the machine if it waits for device approval. Approval can't be taken back here; remove the machine in the dashboard instead.",
				Optional:            true,
			},
			"approved_routes": schema.SetAttribute{
				MarkdownDescription: "Which of the routes the machine offers are approved (subnets like `192.168.1.0/24`, or `0.0.0.0/0` for an exit node).",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"advertised_routes": schema.SetAttribute{
				MarkdownDescription: "The routes the machine offers.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"ipv4": computedString("The machine's address in the network."),
			"fqdn": computedString("The machine's full name."),
			"os":   computedString("Its operating system."),
			"tags": schema.SetAttribute{
				MarkdownDescription: "Its tags (from the auth key it joined with).",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"user": computedString("Who owns it (empty for tagged machines)."),
		},
	}
}

func (r *machineResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.c = clientFrom(req.ProviderData)
}

// fill reads the machine into the model; the arguments left unset stay null.
func (m *machineModel) fill(n client.Machine) {
	m.ID = types.StringValue(n.ID)
	m.MachineID = types.StringValue(n.ID)
	m.Name = types.StringValue(n.Name)
	if !m.KeyExpiryDisabled.IsNull() {
		m.KeyExpiryDisabled = types.BoolValue(n.KeyExpiryDisabled)
	}
	if !m.Approved.IsNull() {
		m.Approved = types.BoolValue(!n.Pending)
	}
	if !m.ApprovedRoutes.IsNull() {
		m.ApprovedRoutes = stringSet(n.ApprovedRoutes)
	}
	m.AdvertisedRoutes = stringSet(n.AdvertisedRoutes)
	m.IPv4 = types.StringValue(n.IPv4)
	m.FQDN = types.StringValue(n.FQDN)
	m.OS = types.StringValue(n.OS)
	m.Tags = stringSet(n.Tags)
	m.User = types.StringValue(n.User)
}

// apply makes the machine match the plan, comparing with how it is now.
func (r *machineResource) apply(ctx context.Context, plan *machineModel) error {
	id := plan.MachineID.ValueString()
	now, err := r.c.Machine(ctx, id)
	if err != nil {
		return err
	}
	var ch client.MachineChange
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() && plan.Name.ValueString() != now.Name {
		v := plan.Name.ValueString()
		ch.Name = &v
	}
	if b := boolPtr(plan.KeyExpiryDisabled); b != nil && *b != now.KeyExpiryDisabled {
		ch.KeyExpiryDisabled = b
	}
	if ch.Name != nil || ch.KeyExpiryDisabled != nil {
		if err := r.c.UpdateMachine(ctx, id, ch); err != nil {
			return err
		}
	}
	if plan.Approved.ValueBool() && now.Pending {
		if err := r.c.ApproveMachine(ctx, id); err != nil {
			return err
		}
	}
	if !plan.ApprovedRoutes.IsNull() && !plan.ApprovedRoutes.IsUnknown() {
		want := []string{}
		if d := plan.ApprovedRoutes.ElementsAs(ctx, &want, false); d.HasError() {
			return errFromDiags(d)
		}
		have := slices.Clone(now.ApprovedRoutes)
		slices.Sort(want)
		slices.Sort(have)
		if !slices.Equal(want, have) {
			if err := r.c.SetApprovedRoutes(ctx, id, want); err != nil {
				return err
			}
		}
	}
	after, err := r.c.Machine(ctx, id)
	if err != nil {
		return err
	}
	plan.fill(after)
	return nil
}

func (r *machineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan machineModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Couldn't update the machine", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *machineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state machineModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	n, err := r.c.Machine(ctx, state.MachineID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx) // removed from the workspace
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Couldn't read the machine", err.Error())
		return
	}
	state.fill(n)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *machineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan machineModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Couldn't update the machine", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete stops managing the machine; it stays in the workspace.
func (r *machineResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {}

// ImportState takes a machine by its ID (`terraform import underpass_machine.web <id>`).
func (r *machineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("machine_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
