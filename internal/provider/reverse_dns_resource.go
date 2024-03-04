// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/kenzo0107/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &reverseDNSResource{}
var _ resource.ResourceWithImportState = &reverseDNSResource{}

func newReverseDNSResource() resource.Resource {
	return &reverseDNSResource{}
}

type reverseDNSResource struct {
	client *sendgrid.Client
}

type reverseDNSResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	IP                    types.String `tfsdk:"ip"`
	RDNS                  types.String `tfsdk:"rdns"`
	Users                 types.Set    `tfsdk:"users"`
	Subdomain             types.String `tfsdk:"subdomain"`
	Domain                types.String `tfsdk:"domain"`
	Valid                 types.Bool   `tfsdk:"valid"`
	Legacy                types.Bool   `tfsdk:"legacy"`
	LastValidationAttempt types.Int64  `tfsdk:"last_validation_attempt"`
	ARecord               types.Object `tfsdk:"a_record"`
}

var aRecordObjectAttribute = map[string]attr.Type{
	"valid": types.BoolType,
	"type":  types.StringType,
	"host":  types.StringType,
	"data":  types.StringType,
}

func (r *reverseDNSResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reverse_dns"
}

func (r *reverseDNSResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides reverseDNS resource.

Reverse DNS (formerly IP Whitelabel) allows mailbox providers to verify the sender of an email by performing a reverse DNS lookup upon receipt of the emails you send.

Reverse DNS is available for [dedicated IP addresses](https://sendgrid.com/docs/ui/account-and-settings/dedicated-ip-addresses/) only.

When setting up reverse DNS, Twilio SendGrid will provide an A Record (address record) for you to add to your DNS records. The A Record maps your sending domain to a dedicated Twilio SendGrid IP address.

A Reverse DNS consists of a subdomain and domain that will be used to generate a reverse DNS record for a given IP address. Once Twilio SendGrid has verified that the appropriate A record for the IP address has been created, the appropriate reverse DNS record for the IP address is generated.

You can also manage your reverse DNS settings in the [Sender Authentication setion of the Twilio SendGrid App](https://app.sendgrid.com/settings/sender_auth).

For more about Reverse DNS, see ["How to set up reverse DNS"](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-reverse-dns/) in the Twilio SendGrid documentation.
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Reverse DNS.",
				Computed:            true,
			},
			"ip": schema.StringAttribute{
				MarkdownDescription: "The IP address that this Reverse DNS was created for.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The root, or sending, domain.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rdns": schema.StringAttribute{
				MarkdownDescription: "The reverse DNS record for the IP address. This points to the Reverse DNS subdomain.",
				Computed:            true,
			},
			"users": schema.SetNestedAttribute{
				MarkdownDescription: "The users who are able to send mail from the IP address.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{
							MarkdownDescription: "The username of a user who can send mail from the IP address.",
							Computed:            true,
						},
						"user_id": schema.Int64Attribute{
							MarkdownDescription: "The ID of a user who can send mail from the IP address.",
							Computed:            true,
						},
					},
				},
			},
			"subdomain": schema.StringAttribute{
				MarkdownDescription: "The subdomain created for this reverse DNS. This is where the rDNS record points.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"valid": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this is a valid Reverse DNS.",
				Computed:            true,
			},
			"legacy": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this Reverse DNS was created using the legacy whitelabel tool. If it is a legacy whitelabel, it will still function, but you'll need to create a new Reverse DNS if you need to update it.",
				Computed:            true,
			},
			"last_validation_attempt": schema.Int64Attribute{
				MarkdownDescription: "A Unix epoch timestamp representing the last time of a validation attempt.",
				Computed:            true,
			},
			"a_record": schema.ObjectAttribute{
				Computed:       true,
				AttributeTypes: aRecordObjectAttribute,
			},
		},
	}
}

func (r *reverseDNSResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *reverseDNSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan reverseDNSResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputCreateReverseDNS{
		IP:     plan.IP.ValueString(),
		Domain: plan.Domain.ValueString(),
	}

	if !plan.Subdomain.IsNull() {
		input.Subdomain = plan.Subdomain.ValueString()
	}

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.CreateReverseDNS(ctx, input)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating reverseDNS",
			fmt.Sprintf("Unable to create reverseDNS, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputCreateReverseDNS)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating reverseDNS",
			"Failed to assert type *sendgrid.OutputCreateReverseDNS",
		)
		return
	}

	plan = reverseDNSResourceModel{
		ID:                    types.StringValue(strconv.FormatInt(o.ID, 10)),
		IP:                    types.StringValue(o.IP),
		RDNS:                  types.StringValue(o.RDNS),
		Subdomain:             types.StringValue(o.Subdomain),
		Domain:                types.StringValue(o.Domain),
		Users:                 convertUsersToSetType(o.Users),
		Valid:                 types.BoolValue(o.Valid),
		Legacy:                types.BoolValue(o.Legacy),
		LastValidationAttempt: types.Int64Value(o.LastValidationAttemptAt),
		ARecord:               newARecord(o.ARecord),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *reverseDNSResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state reverseDNSResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reverseDNSID := state.ID.ValueString()
	id, _ := strconv.ParseInt(reverseDNSID, 10, 64)

	o, err := r.client.GetReverseDNS(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading reverseDNS",
			fmt.Sprintf("Unable to read reverseDNS (id: %v), got error: %s", id, err),
		)
		return
	}

	state = reverseDNSResourceModel{
		ID:                    types.StringValue(strconv.FormatInt(o.ID, 10)),
		IP:                    types.StringValue(o.IP),
		RDNS:                  types.StringValue(o.RDNS),
		Subdomain:             types.StringValue(o.Subdomain),
		Domain:                types.StringValue(o.Domain),
		Users:                 convertUsersToSetType(o.Users),
		Valid:                 types.BoolValue(o.Valid),
		Legacy:                types.BoolValue(o.Legacy),
		LastValidationAttempt: types.Int64Value(o.LastValidationAttemptAt),
		ARecord:               newARecord(o.ARecord),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *reverseDNSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Updating reverseDNS",
		"cannot update reverseDNS, it is immutable",
	)
}

func (r *reverseDNSResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state reverseDNSResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reverseDNSID := state.ID.ValueString()
	id, _ := strconv.ParseInt(reverseDNSID, 10, 64)

	_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return nil, r.client.DeleteReverseDNS(ctx, id)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting reverseDNS",
			fmt.Sprintf("Unable to delete reverseDNS (id: %v), got error: %s", id, err),
		)
		return
	}
}

func (r *reverseDNSResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data reverseDNSResourceModel

	reverseDNSID := req.ID
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	id, _ := strconv.ParseInt(reverseDNSID, 10, 64)

	o, err := r.client.GetReverseDNS(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing reverseDNS",
			fmt.Sprintf("Unable to read reverse DNS, got error: %s", err),
		)
		return
	}

	data = reverseDNSResourceModel{
		ID:                    types.StringValue(strconv.FormatInt(o.ID, 10)),
		IP:                    types.StringValue(o.IP),
		RDNS:                  types.StringValue(o.RDNS),
		Subdomain:             types.StringValue(o.Subdomain),
		Domain:                types.StringValue(o.Domain),
		Users:                 convertUsersToSetType(o.Users),
		Valid:                 types.BoolValue(o.Valid),
		Legacy:                types.BoolValue(o.Legacy),
		LastValidationAttempt: types.Int64Value(o.LastValidationAttemptAt),
		ARecord:               newARecord(o.ARecord),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func convertUsersToSetType(users []*sendgrid.User) basetypes.SetValue {
	var r []attr.Value

	for _, user := range users {
		r = append(r, types.ObjectValueMust(
			map[string]attr.Type{
				"user_id":  types.Int64Type,
				"username": types.StringType,
			},
			map[string]attr.Value{
				"user_id":  types.Int64Value(user.UserID),
				"username": types.StringValue(user.Username),
			},
		))
	}
	return types.SetValueMust(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"user_id":  types.Int64Type,
			"username": types.StringType,
		},
	}, r)
}

func newARecord(record sendgrid.ARecord) basetypes.ObjectValue {
	obj, _ := types.ObjectValue(aRecordObjectAttribute, map[string]attr.Value{
		"valid": types.BoolValue(record.Valid),
		"type":  types.StringValue(record.Type),
		"host":  types.StringValue(record.Host),
		"data":  types.StringValue(record.Data),
	})
	return obj
}
