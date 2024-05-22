// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &enforceTLSResource{}
var _ resource.ResourceWithImportState = &enforceTLSResource{}

func newEnforceTLSResource() resource.Resource {
	return &enforceTLSResource{}
}

type enforceTLSResource struct {
	client *sendgrid.Client
}

type enforceTLSResourceModel struct {
	RequireTLS       types.Bool    `tfsdk:"require_tls"`
	RequireValidCert types.Bool    `tfsdk:"require_valid_cert"`
	Version          types.Float64 `tfsdk:"version"`
}

func (r *enforceTLSResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_enforce_tls"
}

func (r *enforceTLSResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The Enforced TLS settings specify whether or not the recipient of your send is required to support TLS or have a valid certificate. The Enforced TLS endpoint supports retrieving and updating TLS settings.

NOTE: Even if you run the current forced TLS settings acquisition API immediately after updating, the changes may not be reflected.
		`,
		Attributes: map[string]schema.Attribute{
			"require_tls": schema.BoolAttribute{
				MarkdownDescription: "Indicates if you want to require your recipients to support TLS.",
				Optional:            true,
				Computed:            true,
			},
			"require_valid_cert": schema.BoolAttribute{
				MarkdownDescription: "Indicates if you want to require your recipients to have a valid certificate.",
				Optional:            true,
				Computed:            true,
			},
			"version": schema.Float64Attribute{
				MarkdownDescription: "The minimum required TLS certificate version.",
				Optional:            true,
				Computed:            true,
				Default:             float64default.StaticFloat64(1.1),
			},
		},
	}
}

func (r *enforceTLSResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *enforceTLSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan enforceTLSResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputUpdateEnforceTLS{}
	if !plan.RequireTLS.IsNull() {
		input.RequireTLS = plan.RequireTLS.ValueBool()
	}
	if !plan.RequireValidCert.IsNull() {
		input.RequireValidCert = plan.RequireValidCert.ValueBool()
	}
	if !plan.Version.IsNull() {
		input.Version = plan.Version.ValueFloat64()
	}

	o, err := r.client.UpdateEnforceTLS(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating enforceTLS",
			fmt.Sprintf("Unable to update enforceTLS, got error: %s", err),
		)
		return
	}

	plan = enforceTLSResourceModel{
		RequireTLS:       types.BoolValue(o.RequireTLS),
		RequireValidCert: types.BoolValue(o.RequireValidCert),
		Version:          types.Float64Value(o.Version),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *enforceTLSResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state enforceTLSResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	o, err := r.client.GetEnforceTLS(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading enforceTLS",
			fmt.Sprintf("Unable to read enforceTLS, got error: %s", err),
		)
		return
	}

	state = enforceTLSResourceModel{
		RequireTLS:       types.BoolValue(o.RequireTLS),
		RequireValidCert: types.BoolValue(o.RequireValidCert),
		Version:          types.Float64Value(o.Version),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *enforceTLSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state enforceTLSResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputUpdateEnforceTLS{}
	if !data.RequireTLS.IsNull() && data.RequireTLS.ValueBool() != state.RequireTLS.ValueBool() {
		input.RequireTLS = data.RequireTLS.ValueBool()
	}
	if !data.RequireValidCert.IsNull() && data.RequireValidCert.ValueBool() != state.RequireValidCert.ValueBool() {
		input.RequireValidCert = data.RequireValidCert.ValueBool()
	}
	if data.Version.ValueFloat64() != state.Version.ValueFloat64() {
		input.Version = data.Version.ValueFloat64()
	}

	o, err := r.client.UpdateEnforceTLS(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating enforceTLS",
			fmt.Sprintf("Unable to update enforceTLS, got error: %s", err),
		)
		return
	}

	data = enforceTLSResourceModel{
		RequireTLS:       types.BoolValue(o.RequireTLS),
		RequireValidCert: types.BoolValue(o.RequireValidCert),
		Version:          types.Float64Value(o.Version),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *enforceTLSResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state enforceTLSResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *enforceTLSResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data enforceTLSResourceModel

	o, err := r.client.GetEnforceTLS(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing enforceTLS",
			fmt.Sprintf("Unable to read enforceTLS, got error: %s", err),
		)
		return
	}

	data = enforceTLSResourceModel{
		RequireTLS:       types.BoolValue(o.RequireTLS),
		RequireValidCert: types.BoolValue(o.RequireValidCert),
		Version:          types.Float64Value(o.Version),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
