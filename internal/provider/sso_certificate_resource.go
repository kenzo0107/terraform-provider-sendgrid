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
var _ resource.Resource = &ssoCertificateResource{}
var _ resource.ResourceWithImportState = &ssoCertificateResource{}

func newSSOCertificateResource() resource.Resource {
	return &ssoCertificateResource{}
}

type ssoCertificateResource struct {
	client *sendgrid.Client
}

type ssoCertificateResourceModel struct {
	ID                types.String `tfsdk:"id"`
	PublicCertificate types.String `tfsdk:"public_certificate"`
	IntegrationID     types.String `tfsdk:"integration_id"`
	NotBefore         types.Int64  `tfsdk:"not_before"`
	NotAfter          types.Int64  `tfsdk:"not_after"`
}

func (r *ssoCertificateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sso_certificate"
}

func (r *ssoCertificateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides SSO Certificate resource.
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "A unique ID assigned to the certificate by SendGrid.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"public_certificate": schema.StringAttribute{
				MarkdownDescription: "This public certificate allows SendGrid to verify that SAML requests it receives are signed by an IdP that it recognizes.",
				Required:            true,
			},
			"integration_id": schema.StringAttribute{
				MarkdownDescription: "An ID that matches a certificate to a specific IdP integration. This is the id returned by the \"Get All SSO Integrations\" endpoint.",
				Required:            true,
			},
			"not_before": schema.Int64Attribute{
				MarkdownDescription: "A unix timestamp (e.g., 1603915954) that indicates the time before which the certificate is not valid.",
				Computed:            true,
			},
			"not_after": schema.Int64Attribute{
				MarkdownDescription: "A unix timestamp (e.g., 1603915954) that indicates the time after which the certificate is no longer valid.",
				Computed:            true,
			},
		},
	}
}

func (r *ssoCertificateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ssoCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ssoCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputCreateSSOCertificate{
		PublicCertificate: plan.PublicCertificate.ValueString(),
		IntegrationID:     plan.IntegrationID.ValueString(),
		Enabled:           true,
	}

	o, err := r.client.CreateSSOCertificate(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating sso certificate",
			fmt.Sprintf("Unable to create sso certificate, got error: %s", err),
		)
		return
	}

	plan = ssoCertificateResourceModel{
		ID:                types.StringValue(strconv.FormatInt(o.ID, 10)),
		PublicCertificate: types.StringValue(o.PublicCertificate),
		IntegrationID:     types.StringValue(o.IntegrationID),
		NotBefore:         types.Int64Value(o.NotBefore),
		NotAfter:          types.Int64Value(o.NotAfter),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ssoCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ssoCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certificateId := state.ID.ValueString()
	id, _ := strconv.ParseInt(certificateId, 10, 64)

	o, err := r.client.GetSSOCertificate(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading sso certificate",
			fmt.Sprintf("Unable to read sso certificate (id: %v), got error: %s", id, err),
		)
		return
	}

	state = ssoCertificateResourceModel{
		ID:                types.StringValue(strconv.FormatInt(o.ID, 10)),
		PublicCertificate: types.StringValue(o.PublicCertificate),
		IntegrationID:     types.StringValue(o.IntegrationID),
		NotBefore:         types.Int64Value(o.NotBefore),
		NotAfter:          types.Int64Value(o.NotAfter),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// newUpdateSSOCertificateInput builds the PATCH body for an SSO certificate
// update from the planned values.
//
// Enabled has no omitempty in the client library, so it must be set
// explicitly or the request would send false. The resource always creates
// certificates enabled and exposes no enabled attribute, so it is pinned to
// true here as well.
func newUpdateSSOCertificateInput(plan ssoCertificateResourceModel) *sendgrid.InputUpdateSSOCertificate {
	return &sendgrid.InputUpdateSSOCertificate{
		PublicCertificate: plan.PublicCertificate.ValueString(),
		IntegrationID:     plan.IntegrationID.ValueString(),
		Enabled:           true,
	}
}

func (r *ssoCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state ssoCertificateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := newUpdateSSOCertificateInput(data)

	certificateId := state.ID.ValueString()
	id, _ := strconv.ParseInt(certificateId, 10, 64)

	o, err := r.client.UpdateSSOCertificate(ctx, id, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating sso certificate",
			fmt.Sprintf("Unable to sso certificate (id: %s), got error: %s", certificateId, err),
		)
		return
	}

	data = ssoCertificateResourceModel{
		ID:                types.StringValue(strconv.FormatInt(o.ID, 10)),
		PublicCertificate: types.StringValue(o.PublicCertificate),
		IntegrationID:     types.StringValue(o.IntegrationID),
		NotBefore:         types.Int64Value(o.NotBefore),
		NotAfter:          types.Int64Value(o.NotAfter),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ssoCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ssoCertificateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certificateId := state.ID.ValueString()
	id, _ := strconv.ParseInt(certificateId, 10, 64)
	err := r.client.DeleteSSOCertificate(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting sso certificate",
			fmt.Sprintf("Unable to delete sso certificate (id: %v), got error: %s", id, err),
		)
		return
	}
}

func (r *ssoCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data ssoCertificateResourceModel

	certificateId := req.ID
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	id, _ := strconv.ParseInt(certificateId, 10, 64)
	o, err := r.client.GetSSOCertificate(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing sso certificate",
			fmt.Sprintf("Unable to read sso certificate, got error: %s", err),
		)
		return
	}

	data = ssoCertificateResourceModel{
		ID:                types.StringValue(strconv.FormatInt(o.ID, 10)),
		PublicCertificate: types.StringValue(o.PublicCertificate),
		IntegrationID:     types.StringValue(o.IntegrationID),
		NotBefore:         types.Int64Value(o.NotBefore),
		NotAfter:          types.Int64Value(o.NotAfter),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
