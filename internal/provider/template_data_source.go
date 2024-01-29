// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &templateDataSource{}
	_ datasource.DataSourceWithConfigure = &templateDataSource{}
)

func newTemplateDataSource() datasource.DataSource {
	return &templateDataSource{}
}

type templateDataSource struct {
	client *sendgrid.Client
}

type templateDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Generation types.String `tfsdk:"generation"`
}

func (d *templateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template"
}

func (d *templateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (d *templateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides template resource.

An HTML template that can establish a consistent design for [transactional emails](https://sendgrid.com/use-cases/transactional-email/).

Each parent account, as well as each Subuser, can create up to 300 different transactional templates. Templates are specific to the parent account or Subuser, meaning templates created on a parent account will not be accessible from the parent's Subuser accounts.

Transactional templates are templates created specifically for transactional email and are not to be confused with [Marketing Campaigns designs](https://sendgrid.com/docs/ui/sending-email/working-with-marketing-campaigns-email-designs/). For more information about transactional templates, please see our [Dynamic Transactional Templates documentation](https://sendgrid.com/docs/ui/sending-email/how-to-send-an-email-with-dynamic-transactional-templates/).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the transactional template.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name for the transactional template.",
				Computed:            true,
			},
			"generation": schema.StringAttribute{
				MarkdownDescription: "Defines the generation of the template.",
				Computed:            true,
			},
		},
	}
}

func (d *templateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s templateDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := s.ID.ValueString()
	o, err := d.client.GetTemplate(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading template",
			fmt.Sprintf("Unable to get template, got error: %s", err),
		)
		return
	}
	if o.ID == "" {
		resp.Diagnostics.AddError(
			"Reading template",
			fmt.Sprintf("Unable to get template: couldn't find resource (id: %s)", id),
		)
		return
	}

	s.ID = types.StringValue(o.ID)
	s.Name = types.StringValue(o.Name)
	s.Generation = types.StringValue(o.Generation)

	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
