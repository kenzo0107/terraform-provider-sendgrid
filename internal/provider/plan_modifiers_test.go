// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// fakeRawState returns a non-null tftypes value indicating an existing
// resource. The plan modifier checks State.Raw.IsNull() to short-circuit on
// resource creation; this helper ensures we are exercising the update path.
func fakeRawState(t *testing.T) tftypes.Value {
	t.Helper()
	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{"placeholder": tftypes.String},
	}, map[string]tftypes.Value{
		"placeholder": tftypes.NewValue(tftypes.String, "exists"),
	})
}

func TestRequiresReplaceIfStateNotNullString(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stateValue           types.String
		planValue            types.String
		expectRequiresReplace bool
	}{
		"null state, set plan (post-import absorption) should not replace": {
			stateValue:            types.StringNull(),
			planValue:             types.StringValue("new-value"),
			expectRequiresReplace: false,
		},
		"set state, changed plan should replace": {
			stateValue:            types.StringValue("old-value"),
			planValue:             types.StringValue("new-value"),
			expectRequiresReplace: true,
		},
		"set state, same plan should not replace": {
			stateValue:            types.StringValue("same-value"),
			planValue:             types.StringValue("same-value"),
			expectRequiresReplace: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := planmodifier.StringRequest{
				Path:       path.Root("password"),
				StateValue: tc.stateValue,
				PlanValue:  tc.planValue,
				State: tfsdk.State{
					Raw: fakeRawState(t),
				},
				Plan: tfsdk.Plan{
					Raw: fakeRawState(t),
				},
			}
			resp := &planmodifier.StringResponse{
				PlanValue: tc.planValue,
			}

			requiresReplaceIfStateNotNullString().PlanModifyString(context.Background(), req, resp)

			if resp.RequiresReplace != tc.expectRequiresReplace {
				t.Fatalf("expected RequiresReplace=%v, got %v", tc.expectRequiresReplace, resp.RequiresReplace)
			}
		})
	}
}

func TestRequiresReplaceIfStateNotNullInt64(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stateValue           types.Int64
		planValue            types.Int64
		expectRequiresReplace bool
	}{
		"null state, set plan (post-import absorption) should not replace": {
			stateValue:            types.Int64Null(),
			planValue:             types.Int64Value(1),
			expectRequiresReplace: false,
		},
		"set state, changed plan should replace": {
			stateValue:            types.Int64Value(1),
			planValue:             types.Int64Value(2),
			expectRequiresReplace: true,
		},
		"set state, same plan should not replace": {
			stateValue:            types.Int64Value(1),
			planValue:             types.Int64Value(1),
			expectRequiresReplace: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := planmodifier.Int64Request{
				Path:       path.Root("password_wo_version"),
				StateValue: tc.stateValue,
				PlanValue:  tc.planValue,
				State: tfsdk.State{
					Raw: fakeRawState(t),
				},
				Plan: tfsdk.Plan{
					Raw: fakeRawState(t),
				},
			}
			resp := &planmodifier.Int64Response{
				PlanValue: tc.planValue,
			}

			requiresReplaceIfStateNotNullInt64().PlanModifyInt64(context.Background(), req, resp)

			if resp.RequiresReplace != tc.expectRequiresReplace {
				t.Fatalf("expected RequiresReplace=%v, got %v", tc.expectRequiresReplace, resp.RequiresReplace)
			}
		})
	}
}

// Ensure types.String / types.Int64 satisfy attr.Value (compile-time check; avoids unused import lint).
var (
	_ attr.Value = types.StringNull()
	_ attr.Value = types.Int64Null()
)
