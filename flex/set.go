package flex

import (
	"context"
	"fmt"
	"strings"

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

func QuoteAndJoin(items []string) string {
	var quoted []string
	for _, v := range items {
		quoted = append(quoted, fmt.Sprintf("`%s`", v))
	}
	return strings.Join(quoted, ", ")
}

func DiffStringList(ctx context.Context, oldList, newList types.List) (added, removed []string, err error) {
	var oldValues []string
	var newValues []string

	// List -> []string
	diags := oldList.ElementsAs(ctx, &oldValues, false)
	if diags.HasError() {
		return nil, nil, fmt.Errorf("failed to read old list")
	}

	diags = newList.ElementsAs(ctx, &newValues, false)
	if diags.HasError() {
		return nil, nil, fmt.Errorf("failed to read new list")
	}

	oldSet := make(map[string]struct{})
	newSet := make(map[string]struct{})

	for _, v := range oldValues {
		oldSet[v] = struct{}{}
	}

	for _, v := range newValues {
		newSet[v] = struct{}{}
	}

	for v := range oldSet {
		if _, ok := newSet[v]; !ok {
			added = append(added, v)
		}
	}

	for v := range newSet {
		if _, ok := oldSet[v]; !ok {
			removed = append(removed, v)
		}
	}

	return added, removed, nil
}
