// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
