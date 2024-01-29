// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &unsubscribeGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &unsubscribeGroupDataSource{}
)

func newUnsubscribeGroupDataSource() datasource.DataSource {
	return &unsubscribeGroupDataSource{}
}

type unsubscribeGroupDataSource struct {
	client *sendgrid.Client
}

type unsubscribeGroupDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsDefault   types.Bool   `tfsdk:"is_default"`
}

func (d *unsubscribeGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_unsubscribe_group"
}

func (d *unsubscribeGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *unsubscribeGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides suppressions unsubscribe group resource.

Suppression groups, or unsubscribe groups, are specific types or categories of emails from which you would like your recipients to be able to unsubscribe. For example: Daily Newsletters, Invoices, and System Alerts are all potential suppression groups.

Visit the main documentation to [learn more about suppression/unsubscribe groups](https://sendgrid.com/docs/ui/sending-email/unsubscribe-groups/).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the unsubscribe group.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of your suppression group.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A brief description of your suppression group.",
				Computed:            true,
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Indicates if you would like this to be your default suppression group.",
				Computed:            true,
			},
		},
	}
}

func (d *unsubscribeGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s unsubscribeGroupDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := s.ID.ValueString()
	id, _ := strconv.ParseInt(groupID, 10, 64)
	o, err := d.client.GetSuppressionGroup(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading unsubscribe group",
			fmt.Sprintf("Unable to get unsubscribe group, got error: %s", err),
		)
		return
	}

	s.ID = types.StringValue(groupID)
	s.Name = types.StringValue(o.Name)
	s.Description = types.StringValue(o.Description)
	s.IsDefault = types.BoolValue(o.IsDefault)

	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
