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
