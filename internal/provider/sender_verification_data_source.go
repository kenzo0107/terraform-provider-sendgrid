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
	_ datasource.DataSource              = &senderVerificationDataSource{}
	_ datasource.DataSourceWithConfigure = &senderVerificationDataSource{}
)

func newSenderVerificationDataSource() datasource.DataSource {
	return &senderVerificationDataSource{}
}

type senderVerificationDataSource struct {
	client *sendgrid.Client
}

type senderVerificationDataSourceModel struct {
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

func (d *senderVerificationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sender_verification"
}

func (d *senderVerificationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *senderVerificationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a Sender Verification resource.

To protect your sending reputation and to uphold legitimate sending behavior, we require customers to verify their Sender Identities.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/ui/sending-email/sender-verification).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the verified sender.",
				Required:            true,
			},
			"nickname": schema.StringAttribute{
				MarkdownDescription: "A label for your sender identity to help you identify it more quickly. This label is not visible to your recipients.",
				Computed:            true,
			},
			"from_email": schema.StringAttribute{
				MarkdownDescription: "This will display to the user as the email address that sent this email. We will send the verification email to the address you enter in this field. If you have not received your verification email after some time, please refer back to the Sender settings and confirm that the \"From\" email is a valid address.",
				Computed:            true,
			},
			"from_name": schema.StringAttribute{
				MarkdownDescription: "This is a user-friendly name that is displayed to your recipient when they receive their email.",
				Computed:            true,
			},
			"reply_to": schema.StringAttribute{
				MarkdownDescription: "If your user hits reply in their email, the reply will go to this address.",
				Computed:            true,
			},
			"reply_to_name": schema.StringAttribute{
				MarkdownDescription: "reply to name",
				Computed:            true,
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "company address",
				Computed:            true,
			},
			"address2": schema.StringAttribute{
				MarkdownDescription: "company address line 2",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "company state",
				Computed:            true,
			},
			"city": schema.StringAttribute{
				MarkdownDescription: "company city",
				Computed:            true,
			},
			"zip": schema.StringAttribute{
				MarkdownDescription: "company zip",
				Computed:            true,
			},
			"country": schema.StringAttribute{
				MarkdownDescription: "company country",
				Computed:            true,
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

func (d *senderVerificationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s senderVerificationDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := s.ID.ValueString()
	verifiedSenderId, _ := strconv.ParseInt(id, 10, 64)
	senders, err := d.client.GetVerifiedSenders(ctx, &sendgrid.InputGetVerifiedSenders{
		ID:    verifiedSenderId,
		Limit: 1,
	})
	if err != nil || len(senders) == 0 {
		resp.Diagnostics.AddError(
			"Reading sender verification",
			fmt.Sprintf("Unable to get verified sender, got error: %s", err),
		)
		return
	}

	o := senders[0]

	s.Nickname = types.StringValue(o.Nickname)
	s.FromEmail = types.StringValue(o.FromEmail)
	s.FromName = types.StringValue(o.FromName)
	s.ReplyTo = types.StringValue(o.ReplyTo)
	s.ReplyToName = types.StringValue(o.ReplyToName)
	s.Address = types.StringValue(o.Address)
	s.Address2 = types.StringValue(o.Address2)
	s.State = types.StringValue(o.State)
	s.City = types.StringValue(o.City)
	s.Zip = types.StringValue(o.Zip)
	s.Country = types.StringValue(o.Country)
	s.Verified = types.BoolValue(o.Verified)
	s.Locked = types.BoolValue(o.Locked)

	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
