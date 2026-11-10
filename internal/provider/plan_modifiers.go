// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// requiresReplaceIfStateNotNullString returns a plan modifier that requires
// resource replacement only when the prior state value is non-null and the
// plan value differs from it.
//
// This is intended for attributes that cannot be retrieved from the upstream
// API (e.g. SendGrid subuser password). After `terraform import`, the state
// value is null even though the resource exists. Treating the transition from
// null state to a user-provided config value as "must be replaced" forces an
// unintended recreation. With this modifier, that transition is absorbed into
// state silently, while genuine value changes after the first apply still
// trigger replacement.
func requiresReplaceIfStateNotNullString() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			if req.StateValue.IsNull() {
				return
			}
			resp.RequiresReplace = true
		},
		"If the value of this attribute changes after it has been set, Terraform will destroy and recreate the resource.",
		"If the value of this attribute changes after it has been set, Terraform will destroy and recreate the resource.",
	)
}

// requiresReplaceIfStateNotNullInt64 is the int64 counterpart of
// requiresReplaceIfStateNotNullString.
func requiresReplaceIfStateNotNullInt64() planmodifier.Int64 {
	return int64planmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.Int64Request, resp *int64planmodifier.RequiresReplaceIfFuncResponse) {
			if req.StateValue.IsNull() {
				return
			}
			resp.RequiresReplace = true
		},
		"If the value of this attribute changes after it has been set, Terraform will destroy and recreate the resource.",
		"If the value of this attribute changes after it has been set, Terraform will destroy and recreate the resource.",
	)
}

// requiresReplaceIfRemovedString returns a plan modifier that requires
// resource replacement only when a value that was set is removed.
func requiresReplaceIfRemovedString() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = !req.StateValue.IsNull() && req.PlanValue.IsNull()
		},
		"If this attribute is removed after it has been set, Terraform will destroy and recreate the resource.",
		"If this attribute is removed after it has been set, Terraform will destroy and recreate the resource.",
	)
}

// useStateForUnknownIfSignatureUnchanged keeps public_key from the prior state
// while signature_enabled stays the same.
func useStateForUnknownIfSignatureUnchanged() planmodifier.String {
	return useStateForUnknownIfSignatureUnchangedModifier{}
}

type useStateForUnknownIfSignatureUnchangedModifier struct{}

func (m useStateForUnknownIfSignatureUnchangedModifier) Description(_ context.Context) string {
	return "Keeps the prior public key unless signature_enabled changes."
}

func (m useStateForUnknownIfSignatureUnchangedModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateForUnknownIfSignatureUnchangedModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() || !req.PlanValue.IsUnknown() {
		return
	}

	var planEnabled, stateEnabled types.Bool
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("signature_enabled"), &planEnabled)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("signature_enabled"), &stateEnabled)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !planEnabled.IsUnknown() && planEnabled.Equal(stateEnabled) {
		resp.PlanValue = req.StateValue
	}
}

// requiresReplaceIfRemovedList is the list counterpart of
// requiresReplaceIfRemovedString.
func requiresReplaceIfRemovedList() planmodifier.List {
	return listplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.ListRequest, resp *listplanmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = !req.StateValue.IsNull() && req.PlanValue.IsNull()
		},
		"If this attribute is removed after it has been set, Terraform will destroy and recreate the resource.",
		"If this attribute is removed after it has been set, Terraform will destroy and recreate the resource.",
	)
}
