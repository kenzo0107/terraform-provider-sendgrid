// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

var _ resource.Resource = &ipAccessManagementAllowListResource{}
var _ resource.ResourceWithImportState = &ipAccessManagementAllowListResource{}
var _ resource.ResourceWithModifyPlan = &ipAccessManagementAllowListResource{}

const ipAccessManagementAllowListID = "ip_access_management_allow_list"

// allowListEntryRegexp matches IPv4 with optional wildcard octets or a CIDR prefix length.
var allowListEntryRegexp = func() *regexp.Regexp {
	octet := `(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d|\*)`
	return regexp.MustCompile(`^` + octet + `(\.` + octet + `){3}(/(3[0-2]|[12]?\d))?$`)
}()

func newIPAccessManagementAllowListResource() resource.Resource {
	return &ipAccessManagementAllowListResource{}
}

type ipAccessManagementAllowListResource struct {
	client *sendgrid.Client
}

type ipAccessManagementAllowListResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	IPs                    types.Set    `tfsdk:"ips"`
	AcknowledgeLockoutRisk types.Bool   `tfsdk:"acknowledge_lockout_risk"`
	RuleIDs                types.Map    `tfsdk:"rule_ids"`
}

func (r *ipAccessManagementAllowListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_access_management_allow_list"
}

func (r *ipAccessManagementAllowListResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manages IP addresses in the [IP Access Management](https://www.twilio.com/docs/sendgrid/ui/account-and-settings/ip-access-management) allow list.

~> **This resource can lock you out of your SendGrid account.** As soon as the allow list holds one IP address, SendGrid restricts access to the **UI, the API and the SMTP relay** to the allowed addresses. An address missing from the list can therefore:

- stop production mail delivery from your application servers (API and SMTP),
- block the machine or CI runner that runs Terraform from any further ` + "`plan`" + ` or ` + "`apply`" + `,
- lock every user out of the SendGrid UI.

The SendGrid documentation describes no way to turn the restriction off through the API once you are locked out. Recovery requires contacting SendGrid support with thorough proof of your identity. Before applying, make sure ` + "`ips`" + ` covers every address that needs access, including the address Terraform itself runs from.

Behavior:

- All new addresses are sent in a single request, so the list never holds only part of the configured addresses.
- On update, new addresses are added before removed addresses are deleted.
- The resource manages only the addresses listed in ` + "`ips`" + `. Addresses added in the UI or by another resource are left untouched. An address in ` + "`ips`" + ` that is already on the allow list is adopted with a warning instead of being added twice.
- Removing an address from ` + "`ips`" + ` or destroying the resource removes it from the allow list, adopted addresses included. Removing the last address does not deny all access: SendGrid accepts the request and leaves the allow list empty, as it was before the first address was added.
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "A fixed identifier for the resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ips": schema.SetAttribute{
				MarkdownDescription: "The IPv4 addresses to allow. An entry can be a single address (`192.0.2.10`), a wildcard (`192.*.*.*`) or CIDR notation (`192.0.2.10/32`). SendGrid stores every entry as CIDR (`192.0.2.10` as `192.0.2.10/32`, `203.0.*.*` as `203.0.0.0/16`); the value written here is kept in state, so this causes no diff.",
				ElementType:         types.StringType,
				Required:            true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(allowListEntryRegexp, "must be an IPv4 address, optionally with wildcard octets or a CIDR prefix length"),
					),
				},
			},
			"acknowledge_lockout_risk": schema.BoolAttribute{
				MarkdownDescription: "Must be `true`. Confirms that you have read the lockout warning above.",
				Required:            true,
				Validators: []validator.Bool{
					boolvalidator.Equals(true),
				},
			},
			"rule_ids": schema.MapAttribute{
				MarkdownDescription: "The allow list rule ID of each managed address, keyed by the address as written in `ips`.",
				ElementType:         types.Int64Type,
				Computed:            true,
			},
		},
	}
}

func (r *ipAccessManagementAllowListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan keeps rule_ids known when the set of addresses does not change.
func (r *ipAccessManagementAllowListResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state ipAccessManagementAllowListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.IPs.Equal(state.IPs) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("rule_ids"), state.RuleIDs)...)
	}
}

func (r *ipAccessManagementAllowListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ipAccessManagementAllowListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ips []string
	resp.Diagnostics.Append(plan.IPs.ElementsAs(ctx, &ips, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	existing, err := r.client.GetAllowedIPs(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating IP access management allow list",
			fmt.Sprintf("Unable to read the allow list, got error: %s", err),
		)
		return
	}

	ruleIDs, diags := r.addIPs(ctx, ips, existing.Result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, allowListModel(ctx, ruleIDs, &resp.Diagnostics))...)
}

func (r *ipAccessManagementAllowListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ipAccessManagementAllowListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ruleIDs := map[string]int64{}
	resp.Diagnostics.Append(state.RuleIDs.ElementsAs(ctx, &ruleIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	o, err := r.client.GetAllowedIPs(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading IP access management allow list",
			fmt.Sprintf("Unable to read the allow list, got error: %s", err),
		)
		return
	}

	present := allowedRuleIDs(o.Result)
	for ip, id := range ruleIDs {
		if !present[id] {
			delete(ruleIDs, ip)
		}
	}

	if len(ruleIDs) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, allowListModel(ctx, ruleIDs, &resp.Diagnostics))...)
}

func (r *ipAccessManagementAllowListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ipAccessManagementAllowListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planned []string
	current := map[string]int64{}
	resp.Diagnostics.Append(plan.IPs.ElementsAs(ctx, &planned, false)...)
	resp.Diagnostics.Append(state.RuleIDs.ElementsAs(ctx, &current, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	existing, err := r.client.GetAllowedIPs(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating IP access management allow list",
			fmt.Sprintf("Unable to read the allow list, got error: %s", err),
		)
		return
	}

	var toAdd []string
	kept := map[string]int64{}
	for _, ip := range planned {
		if id, ok := current[ip]; ok {
			kept[ip] = id
		} else {
			toAdd = append(toAdd, ip)
		}
	}

	added, diags := r.addIPs(ctx, toAdd, existing.Result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	union := map[string]int64{}
	for ip, id := range current {
		union[ip] = id
	}
	for ip, id := range added {
		union[ip] = id
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, allowListModel(ctx, union, &resp.Diagnostics))...)
	if resp.Diagnostics.HasError() {
		return
	}

	for ip, id := range added {
		kept[ip] = id
	}
	var toRemove []int64
	for ip, id := range current {
		if _, ok := kept[ip]; !ok {
			toRemove = append(toRemove, id)
		}
	}
	if err := r.removeRules(ctx, toRemove, kept, existing.Result); err != nil {
		resp.Diagnostics.AddError(
			"Updating IP access management allow list",
			fmt.Sprintf("Unable to remove IPs from the allow list, got error: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, allowListModel(ctx, kept, &resp.Diagnostics))...)
}

func (r *ipAccessManagementAllowListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ipAccessManagementAllowListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ruleIDs := map[string]int64{}
	resp.Diagnostics.Append(state.RuleIDs.ElementsAs(ctx, &ruleIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	existing, err := r.client.GetAllowedIPs(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting IP access management allow list",
			fmt.Sprintf("Unable to read the allow list, got error: %s", err),
		)
		return
	}

	ids := make([]int64, 0, len(ruleIDs))
	for _, id := range ruleIDs {
		ids = append(ids, id)
	}
	if err := r.removeRules(ctx, ids, nil, existing.Result); err != nil {
		resp.Diagnostics.AddError(
			"Deleting IP access management allow list",
			fmt.Sprintf("Unable to remove IPs from the allow list, got error: %s", err),
		)
		return
	}
}

// ImportState adopts the comma-separated addresses, which must already be on the allow list.
func (r *ipAccessManagementAllowListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var ips []string
	for _, ip := range strings.Split(req.ID, ",") {
		if ip = strings.TrimSpace(ip); ip != "" {
			ips = append(ips, ip)
		}
	}
	if len(ips) == 0 {
		resp.Diagnostics.AddError(
			"Importing IP access management allow list",
			"The import ID must be a comma-separated list of the allowed IPs to manage, for example \"192.0.2.10,198.51.100.0/24\".",
		)
		return
	}

	o, err := r.client.GetAllowedIPs(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing IP access management allow list",
			fmt.Sprintf("Unable to read the allow list, got error: %s", err),
		)
		return
	}

	ruleIDs := map[string]int64{}
	for _, ip := range ips {
		rule := findAllowedIP(o.Result, ip)
		if rule == nil {
			resp.Diagnostics.AddError(
				"Importing IP access management allow list",
				fmt.Sprintf("%q is not on the allow list.", ip),
			)
			continue
		}
		ruleIDs[ip] = rule.ID
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, allowListModel(ctx, ruleIDs, &resp.Diagnostics))...)
}

// addIPs adopts addresses already on the allow list and adds the rest in one request.
func (r *ipAccessManagementAllowListResource) addIPs(ctx context.Context, ips []string, existing []*sendgrid.AllowedIP) (map[string]int64, diag.Diagnostics) {
	var diags diag.Diagnostics
	ruleIDs := map[string]int64{}

	var toAdd []string
	for _, ip := range ips {
		if rule := findAllowedIP(existing, ip); rule != nil {
			ruleIDs[ip] = rule.ID
			diags.AddWarning(
				"IP already on the allow list",
				fmt.Sprintf("%q is already on the allow list (rule ID %d). This resource now manages it and removes it when it leaves \"ips\" or the resource is destroyed.", ip, rule.ID),
			)
			continue
		}
		toAdd = append(toAdd, ip)
	}
	if len(toAdd) == 0 {
		return ruleIDs, diags
	}
	sort.Strings(toAdd)

	input := &sendgrid.InputAddIPsToAllowList{}
	for _, ip := range toAdd {
		input.IPs = append(input.IPs, sendgrid.AllowListIP{IP: ip})
	}
	o, err := r.client.AddIPsToAllowList(ctx, input)
	if err != nil {
		diags.AddError(
			"Adding IPs to the allow list",
			fmt.Sprintf("Unable to add IPs to the allow list, got error: %s", err),
		)
		return nil, diags
	}

	added, err := matchAddedIPs(toAdd, o.Result)
	if err != nil {
		diags.AddError(
			"Adding IPs to the allow list",
			fmt.Sprintf("The IPs were added, but the response could not be matched to the configuration: %s. Import the resource to manage them.", err),
		)
		return nil, diags
	}
	for ip, id := range added {
		ruleIDs[ip] = id
	}
	return ruleIDs, diags
}

// removeRules deletes in one request, skipping rules already gone or still used by a kept address.
func (r *ipAccessManagementAllowListResource) removeRules(ctx context.Context, ids []int64, kept map[string]int64, existing []*sendgrid.AllowedIP) error {
	present := allowedRuleIDs(existing)
	keptIDs := map[int64]bool{}
	for _, id := range kept {
		keptIDs[id] = true
	}

	var remove []int64
	for _, id := range ids {
		if present[id] && !keptIDs[id] && !slices.Contains(remove, id) {
			remove = append(remove, id)
		}
	}
	if len(remove) == 0 {
		return nil
	}
	slices.Sort(remove)

	return r.client.RemoveIPsFromAllowList(ctx, &sendgrid.InputRemoveIPsFromAllowList{IDs: remove})
}

func allowListModel(ctx context.Context, ruleIDs map[string]int64, diags *diag.Diagnostics) *ipAccessManagementAllowListResourceModel {
	ips := make([]string, 0, len(ruleIDs))
	for ip := range ruleIDs {
		ips = append(ips, ip)
	}

	ipSet, d := types.SetValueFrom(ctx, types.StringType, ips)
	diags.Append(d...)
	idMap, d := types.MapValueFrom(ctx, types.Int64Type, ruleIDs)
	diags.Append(d...)

	return &ipAccessManagementAllowListResourceModel{
		ID:                     types.StringValue(ipAccessManagementAllowListID),
		IPs:                    ipSet,
		AcknowledgeLockoutRisk: types.BoolValue(true),
		RuleIDs:                idMap,
	}
}

func allowedRuleIDs(rules []*sendgrid.AllowedIP) map[int64]bool {
	ids := map[int64]bool{}
	for _, rule := range rules {
		if rule != nil {
			ids[rule.ID] = true
		}
	}
	return ids
}

// canonicalAllowListIP makes a single address and its /32 form compare equal.
func canonicalAllowListIP(ip string) string {
	return strings.TrimSuffix(strings.TrimSpace(ip), "/32")
}

// findAllowedIP prefers an exact match and falls back to the canonical form.
func findAllowedIP(rules []*sendgrid.AllowedIP, ip string) *sendgrid.AllowedIP {
	for _, rule := range rules {
		if rule != nil && rule.IP == ip {
			return rule
		}
	}
	for _, rule := range rules {
		if rule != nil && canonicalAllowListIP(rule.IP) == canonicalAllowListIP(ip) {
			return rule
		}
	}
	return nil
}

// matchAddedIPs pairs requested addresses with created rules, by position when the API reformats them.
func matchAddedIPs(requested []string, created []*sendgrid.AllowedIP) (map[string]int64, error) {
	ruleIDs := map[string]int64{}
	rest := slices.DeleteFunc(slices.Clone(created), func(rule *sendgrid.AllowedIP) bool { return rule == nil })

	var unmatched []string
	for _, ip := range requested {
		rule := findAllowedIP(rest, ip)
		if rule == nil {
			unmatched = append(unmatched, ip)
			continue
		}
		ruleIDs[ip] = rule.ID
		rest = slices.DeleteFunc(rest, func(other *sendgrid.AllowedIP) bool { return other == rule })
	}

	if len(unmatched) != len(rest) {
		return nil, fmt.Errorf("requested %d IPs %v but the API returned %d rules", len(requested), requested, len(created))
	}
	for i, ip := range unmatched {
		ruleIDs[ip] = rest[i].ID
	}
	return ruleIDs, nil
}
