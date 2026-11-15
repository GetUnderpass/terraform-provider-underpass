package provider

import (
	"context"

	"github.com/getunderpass/terraform-provider-underpass/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The workspace's settings an API token may change. Only the ones set here
// are managed: the rest stay as the dashboard has them. Settings that act on
// people's screens, copies and machines (remote control, shared clipboard,
// file drops, wake, restart, AI agents, terminal) are people's, in the
// dashboard, and API tokens can't change them.

type settingsResource struct{ c *client.Client }

func newSettingsResource() resource.Resource { return &settingsResource{} }

type settingsModel struct {
	ID               types.String `tfsdk:"id"`
	KeyExpiryDays    types.Int64  `tfsdk:"key_expiry_days"`
	KeyExpiryOff     types.Bool   `tfsdk:"key_expiry_off"`
	DeviceApproval   types.Bool   `tfsdk:"device_approval"`
	NetworkLogs      types.Bool   `tfsdk:"network_logs"`
	AutoUpdate       types.Bool   `tfsdk:"auto_update"`
	LanGames         types.Bool   `tfsdk:"lan_games"`
	LanGamesMachines types.List   `tfsdk:"lan_games_machines"`
}

func (r *settingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_settings"
}

func (r *settingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workspace settings. Only the ones you set are managed; the rest stay as they are in the dashboard. One per workspace. " +
			"Destroying it changes nothing: the settings stay as they were last applied. " +
			"Remote control, shared clipboard, file drops, wake, restart, AI agents and terminal settings are changed by people in the dashboard, not API tokens.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"key_expiry_days": schema.Int64Attribute{
				MarkdownDescription: "How many days machines stay signed in before signing in again: 1 to 365.",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.Between(1, 365)},
			},
			"key_expiry_off": schema.BoolAttribute{
				MarkdownDescription: "Machines never have to sign in again.",
				Optional:            true,
			},
			"device_approval": schema.BoolAttribute{
				MarkdownDescription: "New machines wait for an owner or admin to approve them.",
				Optional:            true,
			},
			"network_logs": schema.BoolAttribute{
				MarkdownDescription: "Keep logs of who connects to what (machines report their connections).",
				Optional:            true,
			},
			"auto_update": schema.BoolAttribute{
				MarkdownDescription: "Machines update themselves as soon as a new version is out.",
				Optional:            true,
			},
			"lan_games": schema.BoolAttribute{
				MarkdownDescription: "Carry machines' local-network broadcasts to each other, so LAN games find each other.",
				Optional:            true,
			},
			"lan_games_machines": schema.ListAttribute{
				MarkdownDescription: "Which machines LAN games covers, as selectors (`*`, `tag:gaming`, `machine:<id>`, `user:<email>`). Empty: all of them.",
				ElementType:         types.StringType,
				Optional:            true,
			},
		},
	}
}

func (r *settingsResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.c = clientFrom(req.ProviderData)
}

// change is what to send: only the settings this configuration manages.
func (m settingsModel) change(ctx context.Context) (client.Settings, error) {
	var s client.Settings
	if !m.KeyExpiryDays.IsNull() {
		v := int(m.KeyExpiryDays.ValueInt64())
		s.KeyExpiryDays = &v
	}
	s.KeyExpiryOff = boolPtr(m.KeyExpiryOff)
	s.DeviceApproval = boolPtr(m.DeviceApproval)
	s.NetworkLogs = boolPtr(m.NetworkLogs)
	s.AutoUpdate = boolPtr(m.AutoUpdate)
	s.LanGames = boolPtr(m.LanGames)
	if !m.LanGamesMachines.IsNull() {
		list := []string{}
		if d := m.LanGamesMachines.ElementsAs(ctx, &list, false); d.HasError() {
			return s, errFromDiags(d)
		}
		s.LanGamesMachines = &list
	}
	return s, nil
}

// fill reads back the managed settings (the others stay null).
func (m *settingsModel) fill(now client.SettingsNow) {
	if !m.KeyExpiryDays.IsNull() {
		m.KeyExpiryDays = types.Int64Value(int64(now.KeyExpiryDays))
	}
	keep := func(b types.Bool, v bool) types.Bool {
		if b.IsNull() {
			return b
		}
		return types.BoolValue(v)
	}
	m.KeyExpiryOff = keep(m.KeyExpiryOff, now.KeyExpiryOff)
	m.DeviceApproval = keep(m.DeviceApproval, now.DeviceApproval)
	m.NetworkLogs = keep(m.NetworkLogs, now.NetworkLogs)
	m.AutoUpdate = keep(m.AutoUpdate, now.AutoUpdate)
	m.LanGames = keep(m.LanGames, now.LanGames)
	if !m.LanGamesMachines.IsNull() {
		m.LanGamesMachines = stringList(now.LanGamesMachines)
	}
}

func (r *settingsResource) apply(ctx context.Context, m *settingsModel) error {
	ch, err := m.change(ctx)
	if err != nil {
		return err
	}
	if err := r.c.UpdateSettings(ctx, ch); err != nil {
		return err
	}
	now, err := r.c.Settings(ctx)
	if err != nil {
		return err
	}
	m.fill(now)
	m.ID = types.StringValue("settings")
	return nil
}

func (r *settingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan settingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Couldn't change the settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *settingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state settingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	now, err := r.c.Settings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Couldn't read the settings", err.Error())
		return
	}
	state.fill(now)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *settingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan settingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Couldn't change the settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete leaves the settings as they are.
func (r *settingsResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}
