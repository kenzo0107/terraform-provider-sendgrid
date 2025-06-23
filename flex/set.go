package flex

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ExpandFrameworkStringSet(ctx context.Context, set types.Set) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}

	var vs []string

	if set.ElementsAs(ctx, &vs, false).HasError() {
		return nil
	}

	return vs
}

func ContainsAll(targets []string, items []string) bool {
	// Convert items to a map for faster lookup
	itemSet := make(map[string]bool)
	for _, item := range items {
		itemSet[item] = true
	}

	// Check if each target exists in the itemSet
	for _, target := range targets {
		if !itemSet[target] {
			return false
		}
	}
	return true
}
