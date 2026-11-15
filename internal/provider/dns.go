package provider

import (
	"context"

	"github.com/getunderpass/terraform-provider-underpass/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The workspace's DNS settings, all of them: what isn't set here is cleared.

type dnsResource struct{ c *client.Client }

func newDNSResource() resource.Resource { return &dnsResource{} }

type dnsModel struct {
	ID            types.String `tfsdk:"id"`
	Nameservers   types.List   `tfsdk:"nameservers"`
	SearchDomains types.List   `tfsdk:"search_domains"`
	SplitDNS      types.Map    `tfsdk:"split_dns"`
	Name          types.String `tfsdk:"name"`
	HTTPS         types.Bool   `tfsdk:"https"`
	Domain        types.String `tfsdk:"domain"`
}

func (r *dnsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns"
}

func (r *dnsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The workspace's [DNS settings](https://docs.getunderpass.com/features/easy-names). One per workspace, and it owns all of them: " +
			"what isn't set here is cleared. Destroying it clears them all (machines keep their own DNS, names stay on the shared domain).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"nameservers": schema.ListAttribute{
				MarkdownDescription: "Nameservers that get every DNS query from the workspace's machines. Empty: machines keep their own DNS for other names.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"search_domains": schema.ListAttribute{
				MarkdownDescription: "Domains tried for short names, after the workspace's own.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"split_dns": schema.MapAttribute{
				MarkdownDescription: "Names under a domain go to that domain's own nameservers: `{ \"corp.example.com\" = [\"10.0.0.53\"] }`.",
				ElementType:         types.ListType{ElemType: types.StringType},
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The workspace's own name, making its domain `<name>.net.getunderpass.com`. Not set: the shared domain.",
				Optional:            true,
			},
			"https": schema.BoolAttribute{
				MarkdownDescription: "Let machines get HTTPS certificates for their names.",
				Optional:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The workspace's domain, as machines are named under it.",
				Computed:            true,
			},
		},
	}
}

func (r *dnsResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.c = clientFrom(req.ProviderData)
}

func (m dnsModel) config(ctx context.Context) (client.DNS, error) {
	var d client.DNS
	if !m.Nameservers.IsNull() {
		if diags := m.Nameservers.ElementsAs(ctx, &d.Nameservers, false); diags.HasError() {
			return d, errFromDiags(diags)
		}
	}
	if !m.SearchDomains.IsNull() {
		if diags := m.SearchDomains.ElementsAs(ctx, &d.SearchDomains, false); diags.HasError() {
			return d, errFromDiags(diags)
		}
	}
	if !m.SplitDNS.IsNull() {
		if diags := m.SplitDNS.ElementsAs(ctx, &d.SplitDNS, false); diags.HasError() {
			return d, errFromDiags(diags)
		}
	}
	d.Name = m.Name.ValueString()
	d.HTTPS = m.HTTPS.ValueBool()
	return d, nil
}

// fill reads back what's there, keeping unset attributes null when the
// workspace has nothing for them, so an empty setting isn't a change.
func (m *dnsModel) fill(d client.DNS) {
	if len(d.Nameservers) > 0 || !m.Nameservers.IsNull() {
		m.Nameservers = stringList(d.Nameservers)
	}
	if len(d.SearchDomains) > 0 || !m.SearchDomains.IsNull() {
		m.SearchDomains = stringList(d.SearchDomains)
	}
	if len(d.SplitDNS) > 0 || !m.SplitDNS.IsNull() {
		elems := map[string]attr.Value{}
		for domain, servers := range d.SplitDNS {
			elems[domain] = stringList(servers)
		}
		m.SplitDNS = types.MapValueMust(types.ListType{ElemType: types.StringType}, elems)
	}
	if d.Name != "" || !m.Name.IsNull() {
		m.Name = types.StringValue(d.Name)
	}
	if d.HTTPS || !m.HTTPS.IsNull() {
		m.HTTPS = types.BoolValue(d.HTTPS)
	}
	m.Domain = types.StringValue(d.Domain)
	m.ID = types.StringValue("dns")
}

func (r *dnsResource) apply(ctx context.Context, m *dnsModel) error {
	d, err := m.config(ctx)
	if err != nil {
		return err
	}
	if err := r.c.SetDNS(ctx, d); err != nil {
		return err
	}
	now, err := r.c.DNS(ctx)
	if err != nil {
		return err
	}
	m.fill(now)
	return nil
}

func (r *dnsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Couldn't save the DNS settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *dnsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	now, err := r.c.DNS(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Couldn't read the DNS settings", err.Error())
		return
	}
	state.fill(now)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *dnsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Couldn't save the DNS settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *dnsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	if err := r.c.SetDNS(ctx, client.DNS{}); err != nil {
		resp.Diagnostics.AddError("Couldn't clear the DNS settings", err.Error())
	}
}
