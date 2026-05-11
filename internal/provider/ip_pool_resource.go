// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
	"github.com/kenzo0107/terraform-provider-sendgrid/flex"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ipPoolResource{}
var _ resource.ResourceWithImportState = &ipPoolResource{}

func newIPPoolResource() resource.Resource {
	return &ipPoolResource{}
}

type ipPoolResource struct {
	client *sendgrid.Client
}

type ipPoolResourceModel struct {
	Name types.String `tfsdk:"name"`
	IPs  types.List   `tfsdk:"ips"`
}

func (r *ipPoolResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_pool"
}

func (r *ipPoolResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides an IP Pool resource.

IP Pools allow you to group your dedicated SendGrid IP addresses together. For example, you could create separate pools for your transactional and marketing email.
		`,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the IP pool.",
				Required:            true,
			},
			"ips": schema.ListAttribute{
				MarkdownDescription: "The list of IPs in the pool.",
				ElementType:         types.StringType,
				Required:            true,
			},
		},
	}
}

func (r *ipPoolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ipPoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ipPoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.CreateIPPool(ctx, &sendgrid.InputCreateIPPool{
			Name: plan.Name.ValueString(),
		})
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating ip pool",
			fmt.Sprintf("Unable to create ip pool, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.OutputCreateIPPool)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating ip pool",
			"Failed to assert type *sendgrid.OutputCreateIPPool",
		)
		return
	}

	var ips []string
	diags := plan.IPs.ElementsAs(ctx, &ips, false)
	if diags.HasError() {
		resp.Diagnostics.AddError(
			"Creating ip pool",
			"Failed to read IPs from plan",
		)
		return
	}

	for _, ip := range ips {
		_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
			return r.client.AddIPToPool(ctx, o.Name, &sendgrid.InputAddIPToPool{
				IP: ip,
			})
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Adding IP to pool",
				fmt.Sprintf("Unable to add ip to pool (name: %s, ip: %s), got error: %s", o.Name, ip, err),
			)
			return
		}
	}

	plan = ipPoolResourceModel{
		Name: types.StringValue(o.Name),
		IPs:  plan.IPs,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ipPoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ipPoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := state.Name.ValueString()
	if name == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	o, err := r.client.GetIPPool(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading ip pool",
			fmt.Sprintf("Unable to read ip pool (name: %s), got error: %s", name, err),
		)
		return
	}

	var ipList []attr.Value
	for _, ip := range o.IPs {
		ipList = append(ipList, types.StringValue(ip.IP))
	}

	state = ipPoolResourceModel{
		Name: types.StringValue(o.PoolName),
		IPs:  types.ListValueMust(types.StringType, ipList),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ipPoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state ipPoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oldName := state.Name.ValueString()
	newName := data.Name.ValueString()

	o, err := r.client.UpdateIPPool(ctx, oldName, &sendgrid.InputUpdateIPPool{
		Name: newName,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating ip pool",
			fmt.Sprintf("Unable to update ip pool (name: %s), got error: %s", oldName, err),
		)
		return
	}

	addIPs, removeIPs, err := flex.DiffStringList(ctx, data.IPs, state.IPs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating ip pool",
			fmt.Sprintf("Unable to diff IP lists, got error: %s", err),
		)
		return
	}
	for _, ip := range addIPs {
		_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
			return r.client.AddIPToPool(ctx, o.Name, &sendgrid.InputAddIPToPool{
				IP: ip,
			})
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Adding IP to pool",
				fmt.Sprintf("Unable to add ip to pool (name: %s, ip: %s), got error: %s", o.Name, ip, err),
			)
			return
		}
	}
	for _, ip := range removeIPs {
		_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
			return nil, r.client.RemoveIPFromPool(ctx, o.Name, ip)
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Removing IP from pool",
				fmt.Sprintf("Unable to remove ip from pool (name: %s, ip: %s), got error: %s", o.Name, ip, err),
			)
			return
		}
	}

	data = ipPoolResourceModel{
		Name: types.StringValue(o.Name),
		IPs:  data.IPs,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ipPoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ipPoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := state.Name.ValueString()

	_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return nil, r.client.DeleteIPPool(ctx, name)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting ip pool",
			fmt.Sprintf("Unable to delete ip pool (name: %s), got error: %s", name, err),
		)
		return
	}
}

func (r *ipPoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	name := req.ID

	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)

	o, err := r.client.GetIPPool(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing ip pool",
			fmt.Sprintf("Unable to import ip pool (name: %s), got error: %s", name, err),
		)
		return
	}

	var ipList []attr.Value
	for _, ip := range o.IPs {
		ipList = append(ipList, types.StringValue(ip.IP))
	}

	state := ipPoolResourceModel{
		Name: types.StringValue(o.PoolName),
		IPs:  types.ListValueMust(types.StringType, ipList),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
