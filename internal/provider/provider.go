// Package provider is the Underpass Terraform provider: the workspace's
// auth keys, access rules, settings, DNS and machines, as code.
package provider

import (
	"context"
	"os"

	"github.com/getunderpass/terraform-provider-underpass/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = (*underpassProvider)(nil)

// New returns the provider for a release version ("dev" when built locally).
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &underpassProvider{version: version} }
}

type underpassProvider struct{ version string }

type providerModel struct {
	APIToken types.String `tfsdk:"api_token"`
	APIURL   types.String `tfsdk:"api_url"`
}

func (p *underpassProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "underpass"
	resp.Version = p.version
}

func (p *underpassProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage an [Underpass](https://getunderpass.com) workspace as code: auth keys, access rules, settings, DNS and machines. " +
			"It acts with an API token (Keys → API tokens in the dashboard), which only ever reaches the workspace it was made in.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				MarkdownDescription: "An API token with **Read & write** scope (`tsapi-…`). Can also be set with `UNDERPASS_API_TOKEN`.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "The coordination service. Defaults to `" + client.DefaultURL + "`; set it when you run your own. Can also be set with `UNDERPASS_API_URL`.",
				Optional:            true,
			},
		},
	}
}

func (p *underpassProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	token := os.Getenv("UNDERPASS_API_TOKEN")
	if !cfg.APIToken.IsNull() && !cfg.APIToken.IsUnknown() {
		token = cfg.APIToken.ValueString()
	}
	url := client.DefaultURL
	if v := os.Getenv("UNDERPASS_API_URL"); v != "" {
		url = v
	}
	if !cfg.APIURL.IsNull() && !cfg.APIURL.IsUnknown() && cfg.APIURL.ValueString() != "" {
		url = cfg.APIURL.ValueString()
	}
	if token == "" {
		resp.Diagnostics.AddError("No API token",
			"Set api_token in the provider block or UNDERPASS_API_TOKEN. Make one in the dashboard: Keys → API tokens, scope Read & write.")
		return
	}
	c := client.New(url, token)
	c.UserAgent = "terraform-provider-underpass/" + p.version
	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *underpassProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newAuthKeyResource,
		newACLResource,
		newSettingsResource,
		newDNSResource,
		newMachineResource,
	}
}

func (p *underpassProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newMachinesDataSource,
		newMachineDataSource,
	}
}

// clientFrom is the API client the provider configured, for resources and
// data sources (nil before configuration, during validation).
func clientFrom(data any) *client.Client {
	c, _ := data.(*client.Client)
	return c
}
