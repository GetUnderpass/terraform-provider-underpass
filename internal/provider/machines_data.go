package provider

import (
	"context"
	"strings"
	"time"

	"github.com/getunderpass/terraform-provider-underpass/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Data sources: the workspace's machines, and one machine by name or ID.

type machineData struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	FQDN             types.String `tfsdk:"fqdn"`
	IPv4             types.String `tfsdk:"ipv4"`
	OS               types.String `tfsdk:"os"`
	User             types.String `tfsdk:"user"`
	Tags             types.Set    `tfsdk:"tags"`
	Online           types.Bool   `tfsdk:"online"`
	LastSeen         types.String `tfsdk:"last_seen"`
	ClientVersion    types.String `tfsdk:"client_version"`
	AdvertisedRoutes types.Set    `tfsdk:"advertised_routes"`
	ApprovedRoutes   types.Set    `tfsdk:"approved_routes"`
}

func machineDataOf(n client.Machine) machineData {
	return machineData{
		ID:               types.StringValue(n.ID),
		Name:             types.StringValue(n.Name),
		FQDN:             types.StringValue(n.FQDN),
		IPv4:             types.StringValue(n.IPv4),
		OS:               types.StringValue(n.OS),
		User:             types.StringValue(n.User),
		Tags:             stringSet(n.Tags),
		Online:           types.BoolValue(n.Online),
		LastSeen:         types.StringValue(n.LastSeen.UTC().Format(time.RFC3339)),
		ClientVersion:    types.StringValue(n.ClientVersion),
		AdvertisedRoutes: stringSet(n.AdvertisedRoutes),
		ApprovedRoutes:   stringSet(n.ApprovedRoutes),
	}
}

func machineAttrs(lookup bool) map[string]schema.Attribute {
	s := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: desc, Computed: true}
	}
	set := func(desc string) schema.SetAttribute {
		return schema.SetAttribute{MarkdownDescription: desc, ElementType: types.StringType, Computed: true}
	}
	attrs := map[string]schema.Attribute{
		"id":                s("The machine's ID."),
		"name":              s("Its name."),
		"fqdn":              s("Its full name."),
		"ipv4":              s("Its address in the network."),
		"os":                s("Its operating system."),
		"user":              s("Who owns it (empty for tagged machines)."),
		"tags":              set("Its tags."),
		"online":            schema.BoolAttribute{MarkdownDescription: "Whether it's connected now.", Computed: true},
		"last_seen":         s("When it was last connected (RFC 3339)."),
		"client_version":    s("The Underpass version it runs."),
		"advertised_routes": set("The routes it offers."),
		"approved_routes":   set("The routes approved for it."),
	}
	if lookup {
		attrs["id"] = schema.StringAttribute{MarkdownDescription: "The machine's ID. Give this or `name`.", Optional: true, Computed: true}
		attrs["name"] = schema.StringAttribute{MarkdownDescription: "The machine's name. Give this or `id`.", Optional: true, Computed: true}
	}
	return attrs
}

// --- underpass_machines ---

type machinesDataSource struct{ c *client.Client }

func newMachinesDataSource() datasource.DataSource { return &machinesDataSource{} }

type machinesModel struct {
	Tag      types.String  `tfsdk:"tag"`
	Machines []machineData `tfsdk:"machines"`
}

func (d *machinesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machines"
}

func (d *machinesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The workspace's machines.",
		Attributes: map[string]schema.Attribute{
			"tag": schema.StringAttribute{MarkdownDescription: "Only machines with this tag, like `tag:servers`.", Optional: true},
			"machines": schema.ListNestedAttribute{
				MarkdownDescription: "The machines, by name.",
				Computed:            true,
				NestedObject:        schema.NestedAttributeObject{Attributes: machineAttrs(false)},
			},
		},
	}
}

func (d *machinesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	d.c = clientFrom(req.ProviderData)
}

func (d *machinesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg machinesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	all, err := d.c.Machines(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Couldn't list the machines", err.Error())
		return
	}
	cfg.Machines = []machineData{}
	for _, n := range all {
		if tag := cfg.Tag.ValueString(); tag != "" && !containsFold(n.Tags, tag) {
			continue
		}
		cfg.Machines = append(cfg.Machines, machineDataOf(n))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg)...)
}

func containsFold(list []string, s string) bool {
	for _, x := range list {
		if strings.EqualFold(x, s) {
			return true
		}
	}
	return false
}

// --- underpass_machine ---

type machineDataSource struct{ c *client.Client }

func newMachineDataSource() datasource.DataSource { return &machineDataSource{} }

func (d *machineDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine"
}

func (d *machineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "One machine, by name or ID.",
		Attributes:          machineAttrs(true),
	}
}

func (d *machineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	d.c = clientFrom(req.ProviderData)
}

func (d *machineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg machineData
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, name := cfg.ID.ValueString(), cfg.Name.ValueString()
	if (id == "") == (name == "") {
		resp.Diagnostics.AddError("Give id or name", "Look a machine up by exactly one of id and name.")
		return
	}
	all, err := d.c.Machines(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Couldn't list the machines", err.Error())
		return
	}
	for _, n := range all {
		if (id != "" && n.ID == id) || (name != "" && strings.EqualFold(n.Name, name)) {
			out := machineDataOf(n)
			resp.Diagnostics.Append(resp.State.Set(ctx, out)...)
			return
		}
	}
	resp.Diagnostics.AddError("No such machine", "No machine in this workspace is called "+name+id+".")
}
