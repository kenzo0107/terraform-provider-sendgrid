// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import "slices"

var autoScopes = []string{
	"2fa_exempt",
	"2fa_required",
	"sender_verification_eligible",
	"sender_verification_exempt",
}

func excludeAutoScopes(scopes []string) []string {
	filteredScopes := make([]string, 0, len(scopes))
	for _, s := range scopes {
		if !slices.Contains(autoScopes, s) {
			filteredScopes = append(filteredScopes, s)
		}
	}
	return filteredScopes
}
