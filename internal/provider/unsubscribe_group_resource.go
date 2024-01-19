// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &unsubscribeGroupResource{}
var _ resource.ResourceWithImportState = &unsubscribeGroupResource{}

func newUnsubscribeGroupResource() resource.Resource {
	return &unsubscribeGroupResource{}
}

type unsubscribeGroupResource struct {
	client *sendgrid.Client
}

type unsubscribeGroupResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsDefault   types.Bool   `tfsdk:"is_default"`
}

func (r *unsubscribeGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_unsubscribe_group"
}

func (r *unsubscribeGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a subuser resource.

Suppression groups, or unsubscribe groups, are specific types or categories of emails from which you would like your recipients to be able to unsubscribe. For example: Daily Newsletters, Invoices, and System Alerts are all potential suppression groups.

Visit the main documentation to [learn more about suppression/unsubscribe groups](https://sendgrid.com/docs/ui/sending-email/unsubscribe-groups/).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the unsubscribe group.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of your suppression group.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A brief description of your suppression group.",
				Optional:            true,
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Indicates if you would like this to be your default suppression group.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *unsubscribeGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sendgrid.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sendgrid.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *unsubscribeGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan unsubscribeGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	o, err := r.client.CreateSuppressionGroup(ctx, &sendgrid.InputCreateSuppressionGroup{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		IsDefault:   plan.IsDefault.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating unsubscribe group",
			fmt.Sprintf("Unable to create unsubscribe group, got error: %s", err),
		)
		return
	}

	plan = unsubscribeGroupResourceModel{
		ID:          types.StringValue(strconv.FormatInt(o.ID, 10)),
		Name:        types.StringValue(o.Name),
		Description: types.StringValue(o.Description),
		IsDefault:   types.BoolValue(o.IsDefault),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *unsubscribeGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state unsubscribeGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := state.ID.ValueString()
	id, _ := strconv.ParseInt(groupID, 10, 64)

	o, err := r.client.GetSuppressionGroup(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading unsubscribe group",
			fmt.Sprintf("Unable to read unsubscribe group (id: %v), got error: %s", id, err),
		)
		return
	}

	state = unsubscribeGroupResourceModel{
		ID:          types.StringValue(strconv.FormatInt(o.ID, 10)),
		Name:        types.StringValue(o.Name),
		Description: types.StringValue(o.Description),
		IsDefault:   types.BoolValue(o.IsDefault),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *unsubscribeGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state unsubscribeGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := state.ID.ValueString()
	id, _ := strconv.ParseInt(groupID, 10, 64)

	o, err := r.client.UpdateSuppressionGroup(ctx, id, &sendgrid.InputUpdateSuppressionGroup{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		IsDefault:   data.IsDefault.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating unsubscribe group",
			fmt.Sprintf("Unable to update unsubscribe group (id: %v), got error: %s", id, err),
		)
		return
	}

	data = unsubscribeGroupResourceModel{
		ID:          state.ID,
		Name:        types.StringValue(o.Name),
		Description: types.StringValue(o.Description),
		IsDefault:   types.BoolValue(o.IsDefault),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *unsubscribeGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state unsubscribeGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := state.ID.ValueString()
	id, _ := strconv.ParseInt(groupID, 10, 64)

	if err := r.client.DeleteSuppressionGroup(ctx, id); err != nil {
		resp.Diagnostics.AddError(
			"Deleting unsubscribe group",
			fmt.Sprintf("Unable to delete unsubscribe group (id: %v), got error: %s", id, err),
		)
		return
	}
}

func (r *unsubscribeGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data unsubscribeGroupResourceModel

	groupID := req.ID
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	id, _ := strconv.ParseInt(groupID, 10, 64)

	o, err := r.client.GetSuppressionGroup(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing unsubscribe group",
			fmt.Sprintf("Unable to read unsubscribe group, got error: %s", err),
		)
		return
	}

	data = unsubscribeGroupResourceModel{
		ID:          types.StringValue(strconv.FormatInt(o.ID, 10)),
		Name:        types.StringValue(o.Name),
		Description: types.StringValue(o.Description),
		IsDefault:   types.BoolValue(o.IsDefault),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
