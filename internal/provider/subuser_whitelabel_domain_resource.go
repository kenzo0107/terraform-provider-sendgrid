// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &subuserWhitelabelDomainResource{}
var _ resource.ResourceWithImportState = &subuserWhitelabelDomainResource{}

func newSubuserWhitelabelDomainResource() resource.Resource {
	return &subuserWhitelabelDomainResource{}
}

type subuserWhitelabelDomainResource struct {
	client *sendgrid.Client
}

type subuserWhitelabelDomainResourceModel struct {
	ID       types.String `tfsdk:"id"`
	DomainID types.Int64  `tfsdk:"domain_id"`
	Subuser  types.String `tfsdk:"subuser"`
}

func (r *subuserWhitelabelDomainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subuser_whitelabel_domain"
}

func (r *subuserWhitelabelDomainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Associates an authenticated domain with a single SendGrid subuser.

This resource is a thin wrapper around the SendGrid "associate authenticated domain with a subuser" API.
The parent account must first authenticate and validate the domain before association.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic ID of this association in the form \"<domain_id>:<subuser>\".",
				Computed:            true,
			},
			"domain_id": schema.Int64Attribute{
				MarkdownDescription: "The authenticated domain ID to associate with the subuser.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"subuser": schema.StringAttribute{
				MarkdownDescription: "The subuser username to associate with the authenticated domain.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *subuserWhitelabelDomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *subuserWhitelabelDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data subuserWhitelabelDomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueInt64()
	subuser := data.Subuser.ValueString()

	_, err := r.client.AssociateAuthenticatedDomainWithSubuser(ctx, domainID, &sendgrid.InputAssociateAuthenticatedDomainWithSubuser{
		Username: subuser,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Associating authenticated domain with subuser",
			fmt.Sprintf("Unable to associate domain %d with subuser %q: %s", domainID, subuser, err),
		)
		return
	}

	id := fmt.Sprintf("%d:%s", domainID, subuser)
	data.ID = types.StringValue(id)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *subuserWhitelabelDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data subuserWhitelabelDomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueInt64()
	subuser := data.Subuser.ValueString()

	assoc, err := r.client.GetAuthenticatedDomainAssociatedWithSubuser(ctx, subuser)
	if err != nil {
		// If the API returns an error that clearly indicates the subuser has no
		// associated domain, we can instead call resp.State.RemoveResource(ctx).
		resp.Diagnostics.AddError(
			"Reading domain-subuser association",
			fmt.Sprintf("Unable to get authenticated domain associated with subuser %q: %s", subuser, err),
		)
		return
	}

	if assoc == nil || assoc.ID == 0 || assoc.ID != domainID {
		// Subuser no longer associated with this domain; remove from state.
		resp.State.RemoveResource(ctx)
		return
	}

	// Nothing else to refresh; keep state as-is.
}

func (r *subuserWhitelabelDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes have RequiresReplace, so this should never be called.
	// Terraform will perform Delete + Create when attributes change.
}

func (r *subuserWhitelabelDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data subuserWhitelabelDomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subuser := data.Subuser.ValueString()

	// The API disassociates by subuser name only.
	err := r.client.DisassociateAuthenticatedDomainFromSubuser(ctx, subuser)
	if err != nil {
		// You may choose to ignore "not found" style errors here if the API exposes them.
		resp.Diagnostics.AddError(
			"Disassociating authenticated domain from subuser",
			fmt.Sprintf("Unable to disassociate subuser %q from authenticated domain: %s", subuser, err),
		)
		return
	}
}

func (r *subuserWhitelabelDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two formats:
	//   - "<subuser>" (just the subuser name)
	//   - "<domain_id>:<subuser>"
	rawID := req.ID

	var (
		domainID int64
		subuser  string
		err      error
	)

	// Try to parse "domain_id:subuser"
	if parts := strings.SplitN(rawID, ":", 2); len(parts) == 2 {
		domainID, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				fmt.Sprintf("Unable to parse domain_id from %q: %s. Expected format: <domain_id>:<subuser> or <subuser>.", rawID, err),
			)
			return
		}
		subuser = parts[1]
	} else {
		// Treat the whole ID as subuser; query API for domain
		subuser = rawID
		assoc, err := r.client.GetAuthenticatedDomainAssociatedWithSubuser(ctx, subuser)
		if err != nil {
			resp.Diagnostics.AddError(
				"Importing domain-subuser association",
				fmt.Sprintf("Unable to get authenticated domain associated with subuser %q: %s", subuser, err),
			)
			return
		}
		if assoc == nil || assoc.ID == 0 {
			resp.Diagnostics.AddError(
				"Importing domain-subuser association",
				fmt.Sprintf("No authenticated domain found for subuser %q", subuser),
			)
			return
		}
		domainID = assoc.ID
	}

	// Synthetic ID "<domain_id>:<subuser>"
	syntheticID := fmt.Sprintf("%d:%s", domainID, subuser)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), syntheticID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subuser"), subuser)...)
}
