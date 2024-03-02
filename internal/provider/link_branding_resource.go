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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/kenzo0107/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &linkBrandingResource{}
var _ resource.ResourceWithImportState = &linkBrandingResource{}

func newLinkBrandingResource() resource.Resource {
	return &linkBrandingResource{}
}

type linkBrandingResource struct {
	client *sendgrid.Client
}

type linkBrandingResourceModel struct {
	ID        types.String `tfsdk:"id"`
	UserID    types.Int64  `tfsdk:"user_id"`
	Domain    types.String `tfsdk:"domain"`
	Subdomain types.String `tfsdk:"subdomain"`
	Username  types.String `tfsdk:"username"`
	Default   types.Bool   `tfsdk:"default"`
	Legacy    types.Bool   `tfsdk:"legacy"`
	Valid     types.Bool   `tfsdk:"valid"`
	DNS       types.Set    `tfsdk:"dns"`
}

func (r *linkBrandingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_link_branding"
}

func (r *linkBrandingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a Link Branding resource.

Email link branding (formerly "Link Whitelabel") allows all of the click-tracked links, opens, and images in your emails to be served from your domain rather than sendgrid.net. Spam filters and recipient servers look at the links within emails to determine whether the email looks trustworthy. They use the reputation of the root domain to determine whether the links can be trusted.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/glossary/link-branding).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the branded link.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The root domain of the branded link.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subdomain": schema.StringAttribute{
				MarkdownDescription: "The subdomain used to generate the DNS records for this link branding. This subdomain must be different from the subdomain used for your authenticated domain.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the account that this link branding is associated with.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the user that this link branding is associated with.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"default": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this is the default link branding.",
				Optional:            true,
				Computed:            true,
			},
			"legacy": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this link branding was created using the legacy whitelabel tool. If it is a legacy whitelabel, it will still function, but you'll need to create new link branding if you need to update it.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"valid": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this link branding is valid.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"dns": schema.SetNestedAttribute{
				MarkdownDescription: "The DNS records generated for this link branding.",
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"valid": schema.BoolAttribute{
							MarkdownDescription: "Indicated whether the CName of the DNS is valid or not.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The type of DNS record.",
							Computed:            true,
						},
						"host": schema.StringAttribute{
							MarkdownDescription: "The domain that this DNS record was created for.",
							Computed:            true,
						},
						"data": schema.StringAttribute{
							MarkdownDescription: "The DNS record.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (r *linkBrandingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *linkBrandingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data linkBrandingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := data.Domain.ValueString()
	input := &sendgrid.InputCreateBrandedLink{
		Domain: domain,
	}

	subdomain := data.Subdomain.ValueString()
	if subdomain != "" {
		input.Subdomain = subdomain
	}
	def := data.Default.ValueBool()
	if def {
		input.Default = def
	}

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.CreateBrandedLink(context.TODO(), input)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating branded link",
			fmt.Sprintf("Unable to create branded link, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputCreateBrandedLink)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating branded link",
			"Failed to assert type *sendgrid.OutputCreateBrandedLink",
		)
		return
	}

	id := strconv.FormatInt(o.ID, 10)
	data.ID = types.StringValue(id)
	data.UserID = types.Int64Value(o.UserID)
	data.Domain = types.StringValue(o.Domain)
	data.Subdomain = types.StringValue(o.Subdomain)
	data.Username = types.StringValue(o.Username)
	data.Default = types.BoolValue(o.Default)
	data.Legacy = types.BoolValue(o.Legacy)
	data.Valid = types.BoolValue(o.Valid)
	data.DNS = convertDNSBrandedLinkToSetType(o.DNS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *linkBrandingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data linkBrandingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	linkId, _ := strconv.ParseInt(id, 10, 64)
	o, err := r.client.GetBrandedLink(ctx, linkId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading branded link",
			fmt.Sprintf("Unable to get branded link, got error: %s", err),
		)
		return
	}

	data.UserID = types.Int64Value(o.UserID)
	data.Domain = types.StringValue(o.Domain)
	data.Subdomain = types.StringValue(o.Subdomain)
	data.Username = types.StringValue(o.Username)
	data.Default = types.BoolValue(o.Default)
	data.Legacy = types.BoolValue(o.Legacy)
	data.Valid = types.BoolValue(o.Valid)
	data.DNS = convertDNSBrandedLinkToSetType(o.DNS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *linkBrandingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data linkBrandingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	domainId, _ := strconv.ParseInt(id, 10, 64)

	o, err := r.client.UpdateBrandedLink(ctx, domainId, &sendgrid.InputUpdateBrandedLink{
		Default: data.Default.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating branded link",
			fmt.Sprintf("Unable to update branded link, got error: %s", err),
		)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(o.ID, 10))
	data.UserID = types.Int64Value(o.UserID)
	data.Subdomain = types.StringValue(o.Subdomain)
	data.Domain = types.StringValue(o.Domain)
	data.Username = types.StringValue(o.Username)
	data.Default = types.BoolValue(o.Default)
	data.Legacy = types.BoolValue(o.Legacy)
	data.Valid = types.BoolValue(o.Valid)
	data.DNS = convertDNSBrandedLinkToSetType(o.DNS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *linkBrandingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data linkBrandingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	linkId := data.ID.ValueString()
	id, _ := strconv.ParseInt(linkId, 10, 64)
	_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return nil, r.client.DeleteBrandedLink(ctx, id)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting branded link",
			fmt.Sprintf("Unable to delete branded link (id: %s), got error: %s", linkId, err),
		)
		return
	}
}

func (r *linkBrandingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data linkBrandingResourceModel

	linkId := req.ID

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	id, err := strconv.ParseInt(linkId, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing branded link",
			fmt.Sprintf("Unable to parse int (id: %s), got error: %s", linkId, err),
		)
		return
	}

	o, err := r.client.GetBrandedLink(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing branded link",
			fmt.Sprintf("Unable to get branded link (id: %s), got error: %s", linkId, err),
		)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(o.ID, 10))
	data.UserID = types.Int64Value(o.UserID)
	data.Domain = types.StringValue(o.Domain)
	data.Subdomain = types.StringValue(o.Subdomain)
	data.Username = types.StringValue(o.Username)
	data.Default = types.BoolValue(o.Default)
	data.Legacy = types.BoolValue(o.Legacy)
	data.Valid = types.BoolValue(o.Valid)
	data.DNS = convertDNSBrandedLinkToSetType(o.DNS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func convertDNSBrandedLinkToSetType(dns sendgrid.DNSBrandedLink) (recordsSet basetypes.SetValue) {
	var records []attr.Value

	if dns.DomainCname.Type != "" {
		records = append(records, types.ObjectValueMust(
			map[string]attr.Type{
				"valid": types.BoolType,
				"type":  types.StringType,
				"host":  types.StringType,
				"data":  types.StringType,
			},
			map[string]attr.Value{
				"valid": types.BoolValue(dns.DomainCname.Valid),
				"type":  types.StringValue(dns.DomainCname.Type),
				"host":  types.StringValue(dns.DomainCname.Host),
				"data":  types.StringValue(dns.DomainCname.Data),
			},
		))
	}
	if dns.OwnerCname.Type != "" {
		records = append(records, types.ObjectValueMust(
			map[string]attr.Type{
				"valid": types.BoolType,
				"type":  types.StringType,
				"host":  types.StringType,
				"data":  types.StringType,
			},
			map[string]attr.Value{
				"valid": types.BoolValue(dns.OwnerCname.Valid),
				"type":  types.StringValue(dns.OwnerCname.Type),
				"host":  types.StringValue(dns.OwnerCname.Host),
				"data":  types.StringValue(dns.OwnerCname.Data),
			},
		))
	}

	var recordVariableElemType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"valid": types.BoolType,
			"type":  types.StringType,
			"host":  types.StringType,
			"data":  types.StringType,
		},
	}
	recordsSet = types.SetValueMust(recordVariableElemType, records)
	if len(records) == 0 {
		recordsSet = types.SetNull(recordVariableElemType)
	}

	return recordsSet
}
