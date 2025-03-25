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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &alertResource{}
var _ resource.ResourceWithImportState = &alertResource{}

func newAlertResource() resource.Resource {
	return &alertResource{}
}

type alertResource struct {
	client *sendgrid.Client
}

type alertResourceModel struct {
	ID         types.String `tfsdk:"id"`
	EmailTo    types.String `tfsdk:"email_to"`
	Type       types.String `tfsdk:"type"`
	Frequency  types.String `tfsdk:"frequency"`
	Percentage types.Int64  `tfsdk:"percentage"`
}

func (r *alertResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert"
}

func (r *alertResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Twilio SendGrid's Alerts feature allows you to receive notifications regarding your usage or program statistics from SendGrid at an email address you specify.
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of alert",
				Computed:            true,
			},
			"email_to": schema.StringAttribute{
				MarkdownDescription: "The email address the alert will be sent to. Example: test@example.com",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of alert you want to create. Can be either usage_limit or stats_notification. Example: usage_limit",
				Required:            true,
				Validators: []validator.String{
					stringOneOf(
						"usage_limit",
						"stats_notification",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"frequency": schema.StringAttribute{
				MarkdownDescription: "If the alert is of type stats_notification, this indicates how frequently the stats notifications will be sent. For example, `daily`, `weekly`, or `monthly`.",
				Optional:            true,
				Computed:            true,
			},
			"percentage": schema.Int64Attribute{
				MarkdownDescription: "If the alert is of type usage_limit, this indicates the percentage of email usage that must be reached before the alert will be sent.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
		},
	}
}

func (r *alertResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *alertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := validateAlert(&plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating alert",
			err.Error(),
		)
		return
	}

	// NOTE: Re-execute after the re-executable time has elapsed when a rate limit occurs
	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.CreateAlert(ctx, &sendgrid.InputCreateAlert{
			EmailTo:    plan.EmailTo.ValueString(),
			Type:       plan.Type.ValueString(),
			Frequency:  plan.Frequency.ValueString(),
			Percentage: plan.Percentage.ValueInt64(),
		})
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating alert",
			fmt.Sprintf("Unable to create alert, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputCreateAlert)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating alert",
			"Failed to assert type *sendgrid.OutputCreateAlert",
		)
		return
	}

	plan = alertResourceModel{
		ID:         types.StringValue(strconv.FormatInt(o.ID, 10)),
		EmailTo:    types.StringValue(o.EmailTo),
		Type:       types.StringValue(o.Type),
		Frequency:  types.StringValue(o.Frequency),
		Percentage: types.Int64Value(o.Percentage),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *alertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if id == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	idInt64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading alert",
			fmt.Sprintf("Unable to read alert (id: %s), got error: %s", id, err),
		)
		return
	}

	o, err := r.client.GetAlert(ctx, idInt64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading alert",
			fmt.Sprintf("Unable to read alert (id: %s), got error: %s", id, err),
		)
		return
	}

	state.ID = types.StringValue(id)
	state.EmailTo = types.StringValue(o.EmailTo)
	state.Type = types.StringValue(o.Type)
	state.Frequency = types.StringValue(o.Frequency)
	state.Percentage = types.Int64Value(o.Percentage)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *alertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state alertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	idInt64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating alert",
			fmt.Sprintf("Unable to update alert, got error: %s", err),
		)
		return
	}

	o, err := r.client.UpdateAlert(ctx, idInt64, &sendgrid.InputUpdateAlert{
		EmailTo:    data.EmailTo.ValueString(),
		Frequency:  data.Frequency.ValueString(),
		Percentage: data.Percentage.ValueInt64(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating alert",
			fmt.Sprintf("Unable to update alert, got error: %s", err),
		)
		return
	}

	data = alertResourceModel{
		ID:         types.StringValue(strconv.FormatInt(o.ID, 10)),
		EmailTo:    types.StringValue(o.EmailTo),
		Type:       types.StringValue(o.Type),
		Frequency:  types.StringValue(o.Frequency),
		Percentage: types.Int64Value(o.Percentage),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *alertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	idInt64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting alert",
			fmt.Sprintf("Unable to update alert, got error: %s", err),
		)
		return
	}

	// NOTE: Re-execute after the re-executable time has elapsed when a rate limit occurs
	_, err = retryOnRateLimit(ctx, func() (interface{}, error) {
		return nil, r.client.DeleteAlert(ctx, idInt64)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting alert",
			fmt.Sprintf("Unable to delete alert (id: %s), got error: %s", id, err),
		)
		return
	}
}

func (r *alertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data alertResourceModel

	id := req.ID
	idInt64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing alert",
			fmt.Sprintf("Unable to read alert, got error: %s", err),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	o, err := r.client.GetAlert(ctx, idInt64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing alert",
			fmt.Sprintf("Unable to read alert, got error: %s", err),
		)
		return
	}

	data = alertResourceModel{
		ID:         types.StringValue(id),
		EmailTo:    types.StringValue(o.EmailTo),
		Type:       types.StringValue(o.Type),
		Frequency:  types.StringValue(o.Frequency),
		Percentage: types.Int64Value(o.Percentage),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func validateAlert(r *alertResourceModel) error {
	if r.Type.ValueString() == "stats_notification" {
		if r.Percentage.ValueInt64() > 0 {
			return fmt.Errorf("percentage is only for type usage_limit")
		}
	}

	if r.Type.ValueString() == "usage_limit" {
		if r.Frequency.ValueString() != "" {
			return fmt.Errorf("frequency is only for type stats_notification")
		}
	}

	return nil
}
