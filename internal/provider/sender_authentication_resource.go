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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/kenzo0107/sendgrid"
	"github.com/kenzo0107/terraform-provider-sendgrid/flex"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &senderAuthenticationResource{}
var _ resource.ResourceWithImportState = &senderAuthenticationResource{}

func newSenderAuthenticationResource() resource.Resource {
	return &senderAuthenticationResource{}
}

type senderAuthenticationResource struct {
	client *sendgrid.Client
}

type senderAuthenticationResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	UserID             types.Int64  `tfsdk:"user_id"`
	Domain             types.String `tfsdk:"domain"`
	Subdomain          types.String `tfsdk:"subdomain"`
	Username           types.String `tfsdk:"username"`
	IPs                types.Set    `tfsdk:"ips"`
	Default            types.Bool   `tfsdk:"default"`
	Legacy             types.Bool   `tfsdk:"legacy"`
	CustomDkimSelector types.String `tfsdk:"custom_dkim_selector"`
	DNS                types.Set    `tfsdk:"dns"`
	Valid              types.Bool   `tfsdk:"valid"`
}

func (r *senderAuthenticationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sender_authentication"
}

func (r *senderAuthenticationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a Sender Authentication resource.

Sender authentication refers to the process of showing email providers that SendGrid has your permission to send emails on your behalf.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/glossary/domain-authentication).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the authenticated domain.",
				Computed:            true,
			},
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the user that this domain is associated with.",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Domain being authenticated.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subdomain": schema.StringAttribute{
				MarkdownDescription: "The subdomain to use for this authenticated domain.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username associated with this domain.",
				Computed:            true,
			},
			"ips": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "The IP addresses that will be included in the custom SPF record for this authenticated domain. NOTE: even if it adds the associated IP when executing the domain authentication API, the response returns an empty list of IPs, which causes a difference with the value set by terraform, so IP association/detachment is not supported.",
				Computed:            true,
			},
			"default": schema.BoolAttribute{
				MarkdownDescription: "Whether to use this authenticated domain as the fallback if no authenticated domains match the sender's domain.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"legacy": schema.BoolAttribute{
				MarkdownDescription: "Whether to use this authenticated domain as the fallback if no authenticated domains match the sender's domain.",
				Computed:            true,
			},
			"custom_dkim_selector": schema.StringAttribute{
				MarkdownDescription: "Add a custom DKIM selector. Accepts three letters or numbers.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"valid": schema.BoolAttribute{
				MarkdownDescription: "Indicates if this is a valid authenticated domain.",
				Computed:            true,
			},
			"dns": schema.SetNestedAttribute{
				Computed: true,
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

func (r *senderAuthenticationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *senderAuthenticationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data senderAuthenticationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := data.Domain.ValueString()
	input := &sendgrid.InputAuthenticateDomain{
		Domain: domain,
	}

	subdomain := data.Subdomain.ValueString()
	if subdomain != "" {
		input.Subdomain = subdomain
	}
	username := data.Username.ValueString()
	if username != "" {
		input.Username = username
	}
	ips := flex.ExpandFrameworkStringSet(ctx, data.IPs)
	if len(ips) > 0 {
		input.IPs = ips
	}
	def := data.Default.ValueBool()
	if def {
		input.Default = def
	}
	customDkimSelector := data.CustomDkimSelector.ValueString()
	if customDkimSelector != "" {
		input.CustomDkimSelector = customDkimSelector
	}

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.AuthenticateDomain(context.TODO(), input)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating sender authentication",
			fmt.Sprintf("Unable to authenticate domain, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputAuthenticateDomain)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating sender authentication",
			"Failed to assert type *sendgrid.OutputAuthenticateDomain",
		)
		return
	}

	ipsSet, d := types.SetValueFrom(ctx, types.StringType, o.IPs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.IPs = ipsSet

	id := strconv.FormatInt(o.ID, 10)
	data.ID = types.StringValue(id)
	data.UserID = types.Int64Value(o.UserID)
	data.Domain = types.StringValue(o.Domain)
	data.Subdomain = types.StringValue(o.Subdomain)
	data.Username = types.StringValue(o.Username)
	data.Default = types.BoolValue(o.Default)
	data.Legacy = types.BoolValue(o.Legacy)
	data.Valid = types.BoolValue(o.Valid)
	data.DNS = convertDNSToSetType(o.DNS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *senderAuthenticationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data senderAuthenticationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	domainId, _ := strconv.ParseInt(id, 10, 64)
	o, err := r.client.GetAuthenticatedDomain(ctx, domainId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading sender authentication",
			fmt.Sprintf("Unable to get authenticated domain, got error: %s", err),
		)
		return
	}

	ipsSet, d := types.SetValueFrom(ctx, types.StringType, o.IPs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.IPs = ipsSet

	data.UserID = types.Int64Value(o.UserID)
	data.Domain = types.StringValue(o.Domain)
	data.Subdomain = types.StringValue(o.Subdomain)
	data.Username = types.StringValue(o.Username)
	data.Default = types.BoolValue(o.Default)
	data.Legacy = types.BoolValue(o.Legacy)
	data.Valid = types.BoolValue(o.Valid)
	data.DNS = convertDNSToSetType(o.DNS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *senderAuthenticationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state senderAuthenticationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	domainId, _ := strconv.ParseInt(id, 10, 64)

	o, err := r.client.UpdateDomainAuthentication(ctx, domainId, &sendgrid.InputUpdateDomainAuthentication{
		Default: data.Default.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating sender authentication",
			fmt.Sprintf("Unable to update authenticated domain, got error: %s", err),
		)
		return
	}

	ipsSet, d := types.SetValueFrom(ctx, types.StringType, o.IPs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(o.ID, 10))
	data.UserID = types.Int64Value(o.UserID)
	data.Subdomain = types.StringValue(o.Subdomain)
	data.Domain = types.StringValue(o.Domain)
	data.Username = types.StringValue(o.Username)
	data.IPs = ipsSet
	data.Default = types.BoolValue(o.Default)
	data.Legacy = types.BoolValue(o.Legacy)
	data.Valid = types.BoolValue(o.Valid)
	data.DNS = convertDNSToSetType(o.DNS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *senderAuthenticationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data senderAuthenticationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainId := data.ID.ValueString()
	id, _ := strconv.ParseInt(domainId, 10, 64)
	_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return nil, r.client.DeleteAuthenticatedDomain(ctx, id)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting sender authentication",
			fmt.Sprintf("Unable to delete authenticated domain (id: %s), got error: %s", domainId, err),
		)
		return
	}
}

func (r *senderAuthenticationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data senderAuthenticationResourceModel

	domainId := req.ID

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	id, err := strconv.ParseInt(domainId, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing sender authentication",
			fmt.Sprintf("Unable to parse int (id: %s), got error: %s", domainId, err),
		)
		return
	}

	o, err := r.client.GetAuthenticatedDomain(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing sender authentication",
			fmt.Sprintf("Unable to get authenticated domain (id: %s), got error: %s", domainId, err),
		)
		return
	}

	ipsSet, d := types.SetValueFrom(ctx, types.StringType, o.IPs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(o.ID, 10))
	data.UserID = types.Int64Value(o.UserID)
	data.Domain = types.StringValue(o.Domain)
	data.Subdomain = types.StringValue(o.Subdomain)
	data.Username = types.StringValue(o.Username)
	data.IPs = ipsSet
	data.Default = types.BoolValue(o.Default)
	data.Legacy = types.BoolValue(o.Legacy)
	data.Valid = types.BoolValue(o.Valid)
	data.DNS = convertDNSToSetType(o.DNS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func convertDNSToSetType(dns sendgrid.DNS) (recordsSet basetypes.SetValue) {
	var records []attr.Value

	if dns.MailCname.Type != "" {
		records = append(records, types.ObjectValueMust(
			map[string]attr.Type{
				"valid": types.BoolType,
				"type":  types.StringType,
				"host":  types.StringType,
				"data":  types.StringType,
			},
			map[string]attr.Value{
				"valid": types.BoolValue(dns.MailCname.Valid),
				"type":  types.StringValue(dns.MailCname.Type),
				"host":  types.StringValue(dns.MailCname.Host),
				"data":  types.StringValue(dns.MailCname.Data),
			},
		))
	}
	if dns.Dkim1.Type != "" {
		records = append(records, types.ObjectValueMust(
			map[string]attr.Type{
				"valid": types.BoolType,
				"type":  types.StringType,
				"host":  types.StringType,
				"data":  types.StringType,
			},
			map[string]attr.Value{
				"valid": types.BoolValue(dns.Dkim1.Valid),
				"type":  types.StringValue(dns.Dkim1.Type),
				"host":  types.StringValue(dns.Dkim1.Host),
				"data":  types.StringValue(dns.Dkim1.Data),
			},
		))
	}
	if dns.Dkim2.Type != "" {
		records = append(records, types.ObjectValueMust(
			map[string]attr.Type{
				"valid": types.BoolType,
				"type":  types.StringType,
				"host":  types.StringType,
				"data":  types.StringType,
			},
			map[string]attr.Value{
				"valid": types.BoolValue(dns.Dkim2.Valid),
				"type":  types.StringValue(dns.Dkim2.Type),
				"host":  types.StringValue(dns.Dkim2.Host),
				"data":  types.StringValue(dns.Dkim2.Data),
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
