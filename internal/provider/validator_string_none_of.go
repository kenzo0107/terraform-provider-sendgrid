package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// stringNoneOf rejects a fixed set of values and states why. The generic "must be none of"
// wording tells practitioners which value was rejected but not what goes wrong when it is used,
// which matters when the API accepts the request and discards part of it instead of failing.
func stringNoneOf(reason string, items ...string) validatorStringNoneOf {
	itemMap := map[string]struct{}{}
	for _, i := range items {
		itemMap[i] = struct{}{}
	}
	return validatorStringNoneOf{
		Items:  itemMap,
		Reason: reason,
	}
}

type validatorStringNoneOf struct {
	Items  map[string]struct{}
	Reason string
}

func (v validatorStringNoneOf) keys() (out []string) {
	for k := range v.Items {
		out = append(out, k)
	}
	return
}

func (v validatorStringNoneOf) Description(ctx context.Context) string {
	return fmt.Sprintf("Item must not be one of %s", strings.Join(v.keys(), " "))
}

func (v validatorStringNoneOf) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Item must not be one of `%s`", strings.Join(v.keys(), "` `"))
}

func (v validatorStringNoneOf) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if _, ok := v.Items[req.ConfigValue.ValueString()]; ok {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value provided",
			fmt.Sprintf("%s, got: %s.", v.Reason, req.ConfigValue.ValueString()),
		)
		return
	}
}
