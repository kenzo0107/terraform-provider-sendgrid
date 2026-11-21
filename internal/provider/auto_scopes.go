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

// Declared auto scopes are echoed from the configuration instead of taken from SendGrid's
// response: SendGrid grants only one of each pair (e.g. 2fa_exempt or 2fa_required), so
// trusting the response would leave a permanent diff, and rejecting them would turn every
// addition to autoScopes into a breaking change for configurations that list the new scope.
func withDeclaredAutoScopes(scopes, declared []string) []string {
	result := excludeAutoScopes(scopes)
	for _, s := range declared {
		if slices.Contains(autoScopes, s) && !slices.Contains(result, s) {
			result = append(result, s)
		}
	}
	return result
}
