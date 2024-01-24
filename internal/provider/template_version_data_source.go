// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &templateVersionDataSource{}
	_ datasource.DataSourceWithConfigure = &templateVersionDataSource{}
)

func newTemplateVersionDataSource() datasource.DataSource {
	return &templateVersionDataSource{}
}

type templateVersionDataSource struct {
	client *sendgrid.Client
}

type templateVersionDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	TemplateID           types.String `tfsdk:"template_id"`
	Active               types.Number `tfsdk:"active"`
	Name                 types.String `tfsdk:"name"`
	HTMLContent          types.String `tfsdk:"html_content"`
	PlainContent         types.String `tfsdk:"plain_content"`
	GeneratePlainContent types.Bool   `tfsdk:"generate_plain_content"`
	Subject              types.String `tfsdk:"subject"`
	Editor               types.String `tfsdk:"editor"`
	TestData             types.String `tfsdk:"test_data"`
	ThumbnailURL         types.String `tfsdk:"thumbnail_url"`
}

func (d *templateVersionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template_version"
}

func (d *templateVersionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *templateVersionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a template version resource.

Represents the code for a particular transactional template. Each transactional template can have multiple versions, each version with its own subject and content. Each user can have up to 300 versions across across all templates.

For more information about transactional templates, please see our Transactional Templates documentation. You can also manage your Transactional Templates in the Dynamic Templates section of the Twilio SendGrid App.
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the transactional template version.",
				Required:            true,
			},
			"template_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the transactional template.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name for the transactional template.",
				Computed:            true,
			},
			"subject": schema.StringAttribute{
				MarkdownDescription: "Subject of the new transactional template version. maxLength: 255",
				Computed:            true,
			},
			"active": schema.NumberAttribute{
				MarkdownDescription: "Set the version as the active version associated with the template (0 is inactive, 1 is active). Only one version of a template can be active. The first version created for a template will automatically be set to Active. Allowed Values: 0, 1",
				Computed:            true,
			},
			"html_content": schema.StringAttribute{
				MarkdownDescription: "The HTML content of the version. Maximum of 1048576 bytes allowed.",
				Computed:            true,
			},
			"plain_content": schema.StringAttribute{
				MarkdownDescription: "Text/plain content of the transactional template version. Maximum of 1048576 bytes allowed.",
				Computed:            true,
			},
			"generate_plain_content": schema.BoolAttribute{
				MarkdownDescription: "If true, plain_content is always generated from html_content. If false, plain_content is not altered.",
				Computed:            true,
			},
			"editor": schema.StringAttribute{
				MarkdownDescription: "The editor used in the UI.",
				Computed:            true,
			},
			"test_data": schema.StringAttribute{
				MarkdownDescription: "For dynamic templates only, the mock json data that will be used for template preview and test sends.",
				Computed:            true,
			},
			"thumbnail_url": schema.StringAttribute{
				MarkdownDescription: "A Thumbnail preview of the template's html content.",
				Computed:            true,
			},
		},
	}
}

func (d *templateVersionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var s templateVersionDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	versionID := s.ID.ValueString()
	templateID := s.TemplateID.ValueString()
	o, err := d.client.GetTemplateVersion(ctx, templateID, versionID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading template version",
			fmt.Sprintf("Unable to get template version, got error: %s", err),
		)
		return
	}
	if o.ID == "" {
		resp.Diagnostics.AddError(
			"Reading template version",
			fmt.Sprintf("Unable to get template version: couldn't find resource (template id: %s, version id: %s)", templateID, versionID),
		)
		return
	}

	s.ID = types.StringValue(o.ID)
	s.Name = types.StringValue(o.Name)
	s.Active = types.NumberValue(big.NewFloat(float64(o.Active)))
	s.HTMLContent = types.StringValue(o.HTMLContent)
	s.PlainContent = types.StringValue(o.PlainContent)
	s.GeneratePlainContent = types.BoolValue(o.GeneratePlainContent)
	s.Subject = types.StringValue(o.Subject)
	s.Editor = types.StringValue(o.Editor)
	s.TestData = types.StringValue(o.TestData)
	s.ThumbnailURL = types.StringValue(o.ThumbnailURL)

	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
