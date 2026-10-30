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
var _ resource.Resource = &subuserLinkBrandingResource{}
var _ resource.ResourceWithImportState = &subuserLinkBrandingResource{}

func newSubuserLinkBrandingResource() resource.Resource {
	return &subuserLinkBrandingResource{}
}

type subuserLinkBrandingResource struct {
	client *sendgrid.Client
}

type subuserLinkBrandingResourceModel struct {
	ID      types.String `tfsdk:"id"`
	LinkID  types.Int64  `tfsdk:"link_id"`
	Subuser types.String `tfsdk:"subuser"`
}

func (r *subuserLinkBrandingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subuser_link_branding"
}

func (r *subuserLinkBrandingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Associates a branded link with a single SendGrid subuser.

This resource is a thin wrapper around the SendGrid "associate a branded link with a subuser" API.
The parent account must first create and validate the branded link before association.
A subuser has at most one branded link, and the same branded link can be associated with many subusers.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic ID of this association in the form \"<link_id>:<subuser>\".",
				Computed:            true,
			},
			"link_id": schema.Int64Attribute{
				MarkdownDescription: "The branded link ID to associate with the subuser.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"subuser": schema.StringAttribute{
				MarkdownDescription: "The subuser username to associate with the branded link.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *subuserLinkBrandingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *subuserLinkBrandingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data subuserLinkBrandingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	linkID := data.LinkID.ValueInt64()
	subuser := data.Subuser.ValueString()

	_, err := r.client.AssociateBrandedLinkWithSubuser(ctx, linkID, &sendgrid.InputAssociateBrandedLinkWithSubuser{
		Username: subuser,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Associating branded link with subuser",
			fmt.Sprintf("Unable to associate branded link %d with subuser %q: %s", linkID, subuser, err),
		)
		return
	}

	data.ID = types.StringValue(subuserLinkBrandingID(linkID, subuser))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *subuserLinkBrandingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data subuserLinkBrandingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subuser := data.Subuser.ValueString()

	link, err := r.client.GetSubuserBrandedLink(ctx, subuser)
	if err != nil {
		if isSubuserBrandedLinkNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Reading branded link-subuser association",
			fmt.Sprintf("Unable to get branded link associated with subuser %q: %s", subuser, err),
		)
		return
	}

	if link == nil || link.ID == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	// A subuser associated with another branded link keeps its state, so the plan shows the replacement.
	data.LinkID = types.Int64Value(link.ID)
	data.ID = types.StringValue(subuserLinkBrandingID(link.ID, subuser))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *subuserLinkBrandingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes have RequiresReplace, so this should never be called.
	// Terraform will perform Delete + Create when attributes change.
}

func (r *subuserLinkBrandingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data subuserLinkBrandingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subuser := data.Subuser.ValueString()

	// The API disassociates by subuser name only.
	err := r.client.DisassociateBrandedLinkWithSubuser(ctx, subuser)
	if err != nil && !isSubuserBrandedLinkNotFound(err) {
		resp.Diagnostics.AddError(
			"Disassociating branded link from subuser",
			fmt.Sprintf("Unable to disassociate subuser %q from branded link: %s", subuser, err),
		)
		return
	}
}

func (r *subuserLinkBrandingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support two formats:
	//   - "<subuser>" (just the subuser name)
	//   - "<link_id>:<subuser>"
	rawID := req.ID

	var (
		linkID  int64
		subuser string
		err     error
	)

	// Try to parse "link_id:subuser"
	if parts := strings.SplitN(rawID, ":", 2); len(parts) == 2 {
		linkID, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				fmt.Sprintf("Unable to parse link_id from %q: %s. Expected format: <link_id>:<subuser> or <subuser>.", rawID, err),
			)
			return
		}
		subuser = parts[1]
	} else {
		// Treat the whole ID as subuser; query API for the branded link
		subuser = rawID
		link, err := r.client.GetSubuserBrandedLink(ctx, subuser)
		if err != nil {
			resp.Diagnostics.AddError(
				"Importing branded link-subuser association",
				fmt.Sprintf("Unable to get branded link associated with subuser %q: %s", subuser, err),
			)
			return
		}
		if link == nil || link.ID == 0 {
			resp.Diagnostics.AddError(
				"Importing branded link-subuser association",
				fmt.Sprintf("No branded link found for subuser %q", subuser),
			)
			return
		}
		linkID = link.ID
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), subuserLinkBrandingID(linkID, subuser))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("link_id"), linkID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subuser"), subuser)...)
}

func subuserLinkBrandingID(linkID int64, subuser string) string {
	return fmt.Sprintf("%d:%s", linkID, subuser)
}

// isSubuserBrandedLinkNotFound reports whether SendGrid answered that the subuser has no branded link,
// or that the subuser itself is gone. The client returns these 404s as plain errors without the status code.
func isSubuserBrandedLinkNotFound(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "resource not found") || strings.Contains(msg, "subuser does not exist")
}
