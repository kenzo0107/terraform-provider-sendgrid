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

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &designDataSource{}
	_ datasource.DataSourceWithConfigure = &designDataSource{}
)

func newDesignDataSource() datasource.DataSource {
	return &designDataSource{}
}

type designDataSource struct {
	client *sendgrid.Client
}

type designDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Editor               types.String `tfsdk:"editor"`
	HTMLContent          types.String `tfsdk:"html_content"`
	PlainContent         types.String `tfsdk:"plain_content"`
	Subject              types.String `tfsdk:"subject"`
	Categories           types.Set    `tfsdk:"categories"`
	GeneratePlainContent types.Bool   `tfsdk:"generate_plain_content"`
	ThumbnailURL         types.String `tfsdk:"thumbnail_url"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
	CreatedAt            types.String `tfsdk:"created_at"`
}

func (d *designDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_design"
}

func (d *designDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *designDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a Design data source.

Designs are reusable email layouts that can be used to create marketing campaigns and single sends.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/api-reference/designs-api/).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the design.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the design.",
				Computed:            true,
			},
			"editor": schema.StringAttribute{
				MarkdownDescription: "The editor used in the UI. Allowed values: `code`, `design`.",
				Computed:            true,
			},
			"html_content": schema.StringAttribute{
				MarkdownDescription: "The HTML content of the design.",
				Computed:            true,
			},
			"plain_content": schema.StringAttribute{
				MarkdownDescription: "The plain text content of the design.",
				Computed:            true,
			},
			"subject": schema.StringAttribute{
				MarkdownDescription: "The subject line of the design.",
				Computed:            true,
			},
			"categories": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "The list of categories applied to the design.",
				Computed:            true,
			},
			"generate_plain_content": schema.BoolAttribute{
				MarkdownDescription: "If `true`, `plain_content` is always generated from `html_content`. If `false`, `plain_content` is not altered.",
				Computed:            true,
			},
			"thumbnail_url": schema.StringAttribute{
				MarkdownDescription: "The URL of the thumbnail for the design.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The date and time the design was last updated.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The date and time the design was created.",
				Computed:            true,
			},
		},
	}
}

func (d *designDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s designDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := s.ID.ValueString()

	o, err := d.client.GetDesign(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading design",
			fmt.Sprintf("Unable to get design (id: %s), got error: %s", id, err),
		)
		return
	}

	categories, diags := types.SetValueFrom(ctx, types.StringType, o.Categories)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	s = designDataSourceModel{
		ID:                   types.StringValue(o.ID),
		Name:                 types.StringValue(o.Name),
		Editor:               types.StringValue(o.Editor),
		HTMLContent:          types.StringValue(o.HTMLContent),
		PlainContent:         types.StringValue(o.PlainContent),
		Subject:              types.StringValue(o.Subject),
		Categories:           categories,
		GeneratePlainContent: types.BoolValue(o.GeneratePlainContent),
		ThumbnailURL:         types.StringValue(o.ThumbnailURL),
		UpdatedAt:            types.StringValue(o.UpdatedAt),
		CreatedAt:            types.StringValue(o.CreatedAt),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
