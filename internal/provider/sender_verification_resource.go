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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &senderVerificationResource{}
var _ resource.ResourceWithImportState = &senderVerificationResource{}

func newSenderVerificationResource() resource.Resource {
	return &senderVerificationResource{}
}

type senderVerificationResource struct {
	client *sendgrid.Client
}

type senderVerificationResourceModel struct {
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

func (r *senderVerificationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sender_verification"
}

func (r *senderVerificationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a Sender Verification resource.

To protect your sending reputation and to uphold legitimate sending behavior, we require customers to verify their Sender Identities.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/ui/sending-email/sender-verification).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the verified sender.",
				Computed:            true,
			},
			"nickname": schema.StringAttribute{
				MarkdownDescription: "A label for your sender identity to help you identify it more quickly. This label is not visible to your recipients.",
				Required:            true,
			},
			"from_email": schema.StringAttribute{
				MarkdownDescription: "This will display to the user as the email address that sent this email. We will send the verification email to the address you enter in this field. If you have not received your verification email after some time, please refer back to the Sender settings and confirm that the \"From\" email is a valid address.",
				Required:            true,
			},
			"from_name": schema.StringAttribute{
				MarkdownDescription: "This is a user-friendly name that is displayed to your recipient when they receive their email.",
				Required:            true,
			},
			"reply_to": schema.StringAttribute{
				MarkdownDescription: "If your user hits reply in their email, the reply will go to this address.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"reply_to_name": schema.StringAttribute{
				MarkdownDescription: "reply to name",
				Optional:            true,
				Computed:            true,
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "company address",
				Required:            true,
			},
			"address2": schema.StringAttribute{
				MarkdownDescription: "company address line 2",
				Optional:            true,
				Computed:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "company state",
				Optional:            true,
				Computed:            true,
			},
			"city": schema.StringAttribute{
				MarkdownDescription: "company city",
				Required:            true,
			},
			"zip": schema.StringAttribute{
				MarkdownDescription: "company zip",
				Optional:            true,
				Computed:            true,
			},
			"country": schema.StringAttribute{
				MarkdownDescription: "company country",
				Required:            true,
			},
			"verified": schema.BoolAttribute{
				MarkdownDescription: "verified",
				Computed:            true,
			},
			"locked": schema.BoolAttribute{
				MarkdownDescription: "locked",
				Computed:            true,
			},
		},
	}
}

func (r *senderVerificationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *senderVerificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data senderVerificationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	o, err := r.client.CreateVerifiedSenderRequest(context.TODO(), &sendgrid.InputCreateVerifiedSenderRequest{
		Nickname:    data.Nickname.ValueString(),
		FromEmail:   data.FromEmail.ValueString(),
		FromName:    data.FromName.ValueString(),
		ReplyTo:     data.ReplyTo.ValueString(),
		ReplyToName: data.ReplyToName.ValueString(),
		Address:     data.Address.ValueString(),
		Address2:    data.Address2.ValueString(),
		State:       data.State.ValueString(),
		City:        data.City.ValueString(),
		Zip:         data.Zip.ValueString(),
		Country:     data.Country.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating sender verification",
			fmt.Sprintf("Unable to verified sender, got error: %s", err),
		)
		return
	}

	id := strconv.FormatInt(o.ID, 10)
	data.ID = types.StringValue(id)
	data.Nickname = types.StringValue(o.Nickname)
	data.FromEmail = types.StringValue(o.FromEmail)
	data.FromName = types.StringValue(o.FromName)
	data.ReplyTo = types.StringValue(o.ReplyTo)
	data.ReplyToName = types.StringValue(o.ReplyToName)
	data.Address = types.StringValue(o.Address)
	data.Address2 = types.StringValue(o.Address2)
	data.State = types.StringValue(o.State)
	data.City = types.StringValue(o.City)
	data.Zip = types.StringValue(o.Zip)
	data.Country = types.StringValue(o.Country)
	data.Verified = types.BoolValue(o.Verified)
	data.Locked = types.BoolValue(o.Locked)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *senderVerificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data senderVerificationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	verifiedSenderId, _ := strconv.ParseInt(id, 10, 64)
	senders, err := r.client.GetVerifiedSenders(ctx, &sendgrid.InputGetVerifiedSenders{
		Limit: 1,
		ID:    verifiedSenderId,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading sender verification",
			fmt.Sprintf("Unable to get verified sender, got error: %s", err),
		)
		return
	}
	if len(senders) == 0 {
		resp.Diagnostics.AddError(
			"Importing sender verification",
			fmt.Sprintf("Unable to get verified sender (id: %s), got error: not found", id),
		)
		return
	}

	o := senders[0]

	data.ID = types.StringValue(strconv.FormatInt(o.ID, 10))
	data.Address = types.StringValue(o.Address)
	data.Address2 = types.StringValue(o.Address2)
	data.City = types.StringValue(o.City)
	data.Country = types.StringValue(o.Country)
	data.State = types.StringValue(o.State)
	data.Zip = types.StringValue(o.Zip)
	data.Nickname = types.StringValue(o.Nickname)
	data.FromEmail = types.StringValue(o.FromEmail)
	data.FromName = types.StringValue(o.FromName)
	data.ReplyTo = types.StringValue(o.ReplyTo)
	data.ReplyToName = types.StringValue(o.ReplyToName)
	data.Verified = types.BoolValue(o.Verified)
	data.Locked = types.BoolValue(o.Locked)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *senderVerificationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state senderVerificationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	verifiedSenderId, _ := strconv.ParseInt(id, 10, 64)

	input := &sendgrid.InputUpdateVerifiedSender{}
	if data.Nickname.ValueString() != "" {
		input.Nickname = data.Nickname.ValueString()
	}
	if data.FromEmail.ValueString() != "" {
		input.FromEmail = data.FromEmail.ValueString()
	}
	if data.FromName.ValueString() != "" {
		input.FromName = data.FromName.ValueString()
	}
	if data.ReplyTo.ValueString() != "" {
		input.ReplyTo = data.ReplyTo.ValueString()
	}
	if data.ReplyToName.ValueString() != "" {
		input.ReplyToName = data.ReplyToName.ValueString()
	}
	if data.Address.ValueString() != "" {
		input.Address = data.Address.ValueString()
	}
	if data.Address2.ValueString() != "" {
		input.Address2 = data.Address2.ValueString()
	}
	if data.State.ValueString() != "" {
		input.State = data.State.ValueString()
	}
	if data.City.ValueString() != "" {
		input.City = data.City.ValueString()
	}
	if data.Zip.ValueString() != "" {
		input.Zip = data.Zip.ValueString()
	}
	if data.Country.ValueString() != "" {
		input.Country = data.Country.ValueString()
	}

	o, err := r.client.UpdateVerifiedSender(ctx, verifiedSenderId, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating sender verification",
			fmt.Sprintf("Unable to update verified sender, got error: %s", err),
		)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(o.ID, 10))
	data.Address = types.StringValue(o.Address)
	data.Address2 = types.StringValue(o.Address2)
	data.City = types.StringValue(o.City)
	data.Country = types.StringValue(o.Country)
	data.State = types.StringValue(o.State)
	data.Zip = types.StringValue(o.Zip)
	data.Nickname = types.StringValue(o.Nickname)
	data.FromEmail = types.StringValue(o.FromEmail)
	data.FromName = types.StringValue(o.FromName)
	data.ReplyTo = types.StringValue(o.ReplyTo)
	data.ReplyToName = types.StringValue(o.ReplyToName)
	data.Verified = types.BoolValue(o.Verified)
	data.Locked = types.BoolValue(o.Locked)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *senderVerificationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data senderVerificationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	verifiedSenderId := data.ID.ValueString()
	id, _ := strconv.ParseInt(verifiedSenderId, 10, 64)
	if err := r.client.DeleteVerifiedSender(ctx, id); err != nil {
		resp.Diagnostics.AddError(
			"Deleting sender verification",
			fmt.Sprintf("Unable to delete verified sender (id: %s), got error: %s", verifiedSenderId, err),
		)
		return
	}
}

func (r *senderVerificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data senderVerificationResourceModel

	id := req.ID
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	verifiedSenderId, _ := strconv.ParseInt(id, 10, 64)
	senders, err := r.client.GetVerifiedSenders(ctx, &sendgrid.InputGetVerifiedSenders{
		Limit: 1,
		ID:    verifiedSenderId,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing sender verification",
			fmt.Sprintf("Unable to get verified sender (id: %s), got error: %s", id, err),
		)
		return
	}
	if len(senders) == 0 {
		resp.Diagnostics.AddError(
			"Importing sender verification",
			fmt.Sprintf("Unable to get verified sender (id: %s), got error: not found", id),
		)
		return
	}

	o := senders[0]

	data.ID = types.StringValue(strconv.FormatInt(o.ID, 10))
	data.Address = types.StringValue(o.Address)
	data.Address2 = types.StringValue(o.Address2)
	data.City = types.StringValue(o.City)
	data.Country = types.StringValue(o.Country)
	data.State = types.StringValue(o.State)
	data.Zip = types.StringValue(o.Zip)
	data.Nickname = types.StringValue(o.Nickname)
	data.FromEmail = types.StringValue(o.FromEmail)
	data.FromName = types.StringValue(o.FromName)
	data.ReplyTo = types.StringValue(o.ReplyTo)
	data.ReplyToName = types.StringValue(o.ReplyToName)
	data.Verified = types.BoolValue(o.Verified)
	data.Locked = types.BoolValue(o.Locked)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
