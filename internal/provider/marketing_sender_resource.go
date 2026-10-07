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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

var _ resource.Resource = &marketingSenderResource{}
var _ resource.ResourceWithImportState = &marketingSenderResource{}

func newMarketingSenderResource() resource.Resource {
	return &marketingSenderResource{}
}

type marketingSenderResource struct {
	client *sendgrid.Client
}

type marketingSenderResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Nickname    types.String `tfsdk:"nickname"`
	FromEmail   types.String `tfsdk:"from_email"`
	FromName    types.String `tfsdk:"from_name"`
	ReplyTo     types.String `tfsdk:"reply_to"`
	ReplyToName types.String `tfsdk:"reply_to_name"`
	Address     types.String `tfsdk:"address"`
	Address2    types.String `tfsdk:"address2"`
	State       types.String `tfsdk:"state"`
	City        types.String `tfsdk:"city"`
	Zip         types.String `tfsdk:"zip"`
	Country     types.String `tfsdk:"country"`
	Verified    types.Bool   `tfsdk:"verified"`
	Locked      types.Bool   `tfsdk:"locked"`
}

func newMarketingSenderResourceModel(o *sendgrid.MarketingSender) marketingSenderResourceModel {
	m := marketingSenderResourceModel{
		ID:       types.StringValue(strconv.FormatInt(o.ID, 10)),
		Nickname: types.StringValue(o.Nickname),
		Address:  types.StringValue(o.Address),
		Address2: types.StringValue(o.Address2),
		State:    types.StringValue(o.State),
		City:     types.StringValue(o.City),
		Zip:      types.StringValue(o.Zip),
		Country:  types.StringValue(o.Country),
		Locked:   types.BoolValue(o.Locked),
		Verified: types.BoolValue(o.Verified != nil && o.Verified.Status),
	}
	if o.From != nil {
		m.FromEmail = types.StringValue(o.From.Email)
		m.FromName = types.StringValue(o.From.Name)
	} else {
		m.FromEmail = types.StringValue("")
		m.FromName = types.StringValue("")
	}
	if o.ReplyTo != nil {
		m.ReplyTo = types.StringValue(o.ReplyTo.Email)
		m.ReplyToName = types.StringValue(o.ReplyTo.Name)
	} else {
		m.ReplyTo = types.StringValue("")
		m.ReplyToName = types.StringValue("")
	}
	return m
}

func (r *marketingSenderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_marketing_sender"
}

func (r *marketingSenderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a Marketing Campaigns Sender resource.

Senders are the "From" identities used when sending Marketing Campaigns. If the "From" email address is not on an authenticated domain, SendGrid sends a verification email to that address.

For more detailed information, please see the [SendGrid documentation](https://www.twilio.com/docs/sendgrid/api-reference/marketing-campaigns-senders).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the sender.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"nickname": schema.StringAttribute{
				MarkdownDescription: "A nickname for the sender identity. Not used for sending.",
				Required:            true,
			},
			"from_email": schema.StringAttribute{
				MarkdownDescription: "The email address from which your recipient will receive emails.",
				Required:            true,
			},
			"from_name": schema.StringAttribute{
				MarkdownDescription: "The name appended to the from email field. Typically your name or company name.",
				Required:            true,
			},
			"reply_to": schema.StringAttribute{
				MarkdownDescription: "The email address to which your recipient will reply.",
				Required:            true,
			},
			"reply_to_name": schema.StringAttribute{
				MarkdownDescription: "The name appended to the reply to email field.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "The physical address of the sender identity.",
				Required:            true,
			},
			"address2": schema.StringAttribute{
				MarkdownDescription: "Additional sender identity address information.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The state of the sender identity. SendGrid stores only the first two characters.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"city": schema.StringAttribute{
				MarkdownDescription: "The city of the sender identity.",
				Required:            true,
			},
			"zip": schema.StringAttribute{
				MarkdownDescription: "The zipcode of the sender identity.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"country": schema.StringAttribute{
				MarkdownDescription: "The country of the sender identity.",
				Required:            true,
			},
			"verified": schema.BoolAttribute{
				MarkdownDescription: "Whether the sender identity has been verified.",
				Computed:            true,
			},
			"locked": schema.BoolAttribute{
				MarkdownDescription: "Whether the sender is used in a campaign that is in progress. A locked sender cannot be updated or deleted.",
				Computed:            true,
			},
		},
	}
}

func (r *marketingSenderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := clientFromProviderData(req.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *providerData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *marketingSenderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan marketingSenderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	o, err := r.client.CreateMarketingSender(ctx, &sendgrid.InputCreateMarketingSender{
		Nickname: plan.Nickname.ValueString(),
		From: &sendgrid.MarketingSenderAddress{
			Email: plan.FromEmail.ValueString(),
			Name:  plan.FromName.ValueString(),
		},
		ReplyTo: &sendgrid.MarketingSenderAddress{
			Email: plan.ReplyTo.ValueString(),
			Name:  plan.ReplyToName.ValueString(),
		},
		Address:  plan.Address.ValueString(),
		Address2: plan.Address2.ValueString(),
		State:    plan.State.ValueString(),
		City:     plan.City.ValueString(),
		Zip:      plan.Zip.ValueString(),
		Country:  plan.Country.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating marketing sender",
			fmt.Sprintf("Unable to create marketing sender, got error: %s", err),
		)
		return
	}

	state := newMarketingSenderResourceModel((*sendgrid.MarketingSender)(o))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *marketingSenderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state marketingSenderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	idInt64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading marketing sender",
			fmt.Sprintf("Unable to parse marketing sender id (id: %s), got error: %s", id, err),
		)
		return
	}

	o, err := r.client.GetMarketingSender(ctx, idInt64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading marketing sender",
			fmt.Sprintf("Unable to read marketing sender (id: %s), got error: %s", id, err),
		)
		return
	}

	state = newMarketingSenderResourceModel((*sendgrid.MarketingSender)(o))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *marketingSenderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state marketingSenderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	idInt64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating marketing sender",
			fmt.Sprintf("Unable to parse marketing sender id (id: %s), got error: %s", id, err),
		)
		return
	}

	err = r.updateMarketingSender(ctx, idInt64, &marketingSenderUpdateInput{
		Nickname: plan.Nickname.ValueString(),
		From: marketingSenderUpdateAddress{
			Email: plan.FromEmail.ValueString(),
			Name:  plan.FromName.ValueString(),
		},
		ReplyTo: marketingSenderUpdateAddress{
			Email: plan.ReplyTo.ValueString(),
			Name:  plan.ReplyToName.ValueString(),
		},
		Address:  plan.Address.ValueString(),
		Address2: plan.Address2.ValueString(),
		State:    plan.State.ValueString(),
		City:     plan.City.ValueString(),
		Zip:      plan.Zip.ValueString(),
		Country:  plan.Country.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating marketing sender",
			fmt.Sprintf("Unable to update marketing sender (id: %s), got error: %s", id, err),
		)
		return
	}

	// The PATCH response returns the values before the update, so read the sender again.
	o, err := r.client.GetMarketingSender(ctx, idInt64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating marketing sender",
			fmt.Sprintf("Unable to read marketing sender after update (id: %s), got error: %s", id, err),
		)
		return
	}

	state = newMarketingSenderResourceModel((*sendgrid.MarketingSender)(o))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type marketingSenderUpdateAddress struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type marketingSenderUpdateInput struct {
	Nickname string                       `json:"nickname"`
	From     marketingSenderUpdateAddress `json:"from"`
	ReplyTo  marketingSenderUpdateAddress `json:"reply_to"`
	Address  string                       `json:"address"`
	Address2 string                       `json:"address_2"`
	State    string                       `json:"state"`
	City     string                       `json:"city"`
	Zip      string                       `json:"zip"`
	Country  string                       `json:"country"`
}

// sendgrid.InputUpdateMarketingSender is not used because its omitempty tags drop
// empty strings, which makes it impossible to clear address2, state, zip and reply_to_name.
func (r *marketingSenderResource) updateMarketingSender(ctx context.Context, id int64, input *marketingSenderUpdateInput) error {
	req, err := r.client.NewRequest("PATCH", fmt.Sprintf("/marketing/senders/%d", id), input)
	if err != nil {
		return err
	}
	return r.client.Do(ctx, req, nil)
}

func (r *marketingSenderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state marketingSenderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	idInt64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting marketing sender",
			fmt.Sprintf("Unable to parse marketing sender id (id: %s), got error: %s", id, err),
		)
		return
	}

	if err := r.client.DeleteMarketingSender(ctx, idInt64); err != nil {
		resp.Diagnostics.AddError(
			"Deleting marketing sender",
			fmt.Sprintf("Unable to delete marketing sender (id: %s), got error: %s", id, err),
		)
		return
	}
}

func (r *marketingSenderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
