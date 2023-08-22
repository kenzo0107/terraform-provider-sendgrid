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
