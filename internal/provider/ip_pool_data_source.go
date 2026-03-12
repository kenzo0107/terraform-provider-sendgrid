package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &ipPoolDataSource{}
	_ datasource.DataSourceWithConfigure = &ipPoolDataSource{}
)

func newIPPoolDataSource() datasource.DataSource {
	return &ipPoolDataSource{}
}

type ipPoolDataSource struct {
	client *sendgrid.Client
}

type ipPoolDataSourceModel struct {
	Name types.String `tfsdk:"name"`
	IPs  types.List   `tfsdk:"ips"`
}

func (d *ipPoolDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_pool"
}

func (d *ipPoolDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sendgrid.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *sendgrid.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *ipPoolDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides an IP Pool data source.

IP Pools allow you to group your dedicated SendGrid IP addresses together. For example, you could create separate pools for your transactional and marketing email.
		`,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the IP pool.",
				Required:            true,
			},
			"ips": schema.ListAttribute{
				MarkdownDescription: "The list of IPs in the pool.",
				ElementType:         types.StringType,
				Computed:            true,
			},
		},
	}
}

func (d *ipPoolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s ipPoolDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := s.Name.ValueString()

	o, err := d.client.GetIPPool(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading ip pool",
			fmt.Sprintf("Unable to get ip pool (name: %s), got error: %s", name, err),
		)
		return
	}

	if o == nil {
		resp.Diagnostics.AddError(
			"Reading ip pool",
			fmt.Sprintf("Not found ip pool (name: %s)", name),
		)
		return
	}

	var listIPs []attr.Value
	for _, ip := range o.IPs {
		listIPs = append(listIPs, types.StringValue(ip.IP))
	}

	s = ipPoolDataSourceModel{
		Name: types.StringValue(o.PoolName),
		IPs:  types.ListValueMust(types.StringType, listIPs),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
