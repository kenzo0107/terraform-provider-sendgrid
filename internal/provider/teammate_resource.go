// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/kenzo0107/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &teammateResource{}
var _ resource.ResourceWithImportState = &teammateResource{}

var autoScopes = []string{
	"2fa_exempt",
	"2fa_required",
	"sender_verification_exempt",
	"sender_verification_eligible",
}

// see: https://api.sendgrid.com/v3/scopes
var validScopes = []string{
	"access_settings.activity.read",
	"access_settings.whitelist.create",
	"access_settings.whitelist.delete",
	"access_settings.whitelist.read",
	"access_settings.whitelist.update",
	"alerts.create",
	"alerts.delete",
	"alerts.read",
	"alerts.update",
	"api_keys.create",
	"api_keys.delete",
	"api_keys.read",
	"api_keys.update",
	"asm.groups.create",
	"asm.groups.delete",
	"asm.groups.read",
	"asm.groups.suppressions.create",
	"asm.groups.suppressions.delete",
	"asm.groups.suppressions.read",
	"asm.groups.suppressions.update",
	"asm.groups.update",
	"asm.suppressions.global.create",
	"asm.suppressions.global.delete",
	"asm.suppressions.global.read",
	"asm.suppressions.global.update",
	"billing.create",
	"billing.delete",
	"billing.read",
	"billing.update",
	"browsers.stats.read",
	"categories.create",
	"categories.delete",
	"categories.read",
	"categories.stats.read",
	"categories.stats.sums.read",
	"categories.update",
	"clients.desktop.stats.read",
	"clients.phone.stats.read",
	"clients.stats.read",
	"clients.tablet.stats.read",
	"clients.webmail.stats.read",
	"credentials.create",
	"credentials.delete",
	"credentials.read",
	"credentials.update",
	"design_library.create",
	"design_library.delete",
	"design_library.read",
	"design_library.update",
	"devices.stats.read",
	"di.bounce_block_classification.read",
	"email_testing.read",
	"email_testing.write",
	"geo.stats.read",
	"ips.assigned.read",
	"ips.create",
	"ips.delete",
	"ips.pools.create",
	"ips.pools.delete",
	"ips.pools.ips.create",
	"ips.pools.ips.delete",
	"ips.pools.ips.read",
	"ips.pools.ips.update",
	"ips.pools.read",
	"ips.pools.update",
	"ips.read",
	"ips.update",
	"ips.warmup.create",
	"ips.warmup.delete",
	"ips.warmup.read",
	"ips.warmup.update",
	"mail.batch.create",
	"mail.batch.delete",
	"mail.batch.read",
	"mail.batch.update",
	"mail.send",
	"mail_settings.address_whitelist.create",
	"mail_settings.address_whitelist.delete",
	"mail_settings.address_whitelist.read",
	"mail_settings.address_whitelist.update",
	"mail_settings.bcc.create",
	"mail_settings.bcc.delete",
	"mail_settings.bcc.read",
	"mail_settings.bcc.update",
	"mail_settings.bounce_purge.create",
	"mail_settings.bounce_purge.delete",
	"mail_settings.bounce_purge.read",
	"mail_settings.bounce_purge.update",
	"mail_settings.footer.create",
	"mail_settings.footer.delete",
	"mail_settings.footer.read",
	"mail_settings.footer.update",
	"mail_settings.forward_bounce.create",
	"mail_settings.forward_bounce.delete",
	"mail_settings.forward_bounce.read",
	"mail_settings.forward_bounce.update",
	"mail_settings.forward_spam.create",
	"mail_settings.forward_spam.delete",
	"mail_settings.forward_spam.read",
	"mail_settings.forward_spam.update",
	"mail_settings.plain_content.create",
	"mail_settings.plain_content.delete",
	"mail_settings.plain_content.read",
	"mail_settings.plain_content.update",
	"mail_settings.read",
	"mail_settings.spam_check.create",
	"mail_settings.spam_check.delete",
	"mail_settings.spam_check.read",
	"mail_settings.spam_check.update",
	"mail_settings.template.create",
	"mail_settings.template.delete",
	"mail_settings.template.read",
	"mail_settings.template.update",
	"mailbox_providers.stats.read",
	"marketing.automation.read",
	"marketing.read",
	"messages.read",
	"newsletter.create",
	"newsletter.delete",
	"newsletter.read",
	"newsletter.update",
	"partner_settings.new_relic.create",
	"partner_settings.new_relic.delete",
	"partner_settings.new_relic.read",
	"partner_settings.new_relic.update",
	"partner_settings.read",
	"partner_settings.sendwithus.create",
	"partner_settings.sendwithus.delete",
	"partner_settings.sendwithus.read",
	"partner_settings.sendwithus.update",
	"recipients.erasejob.create",
	"recipients.erasejob.read",
	"sender_verification_eligible",
	"signup.trigger_confirmation",
	"sso.settings.create",
	"sso.settings.delete",
	"sso.settings.read",
	"sso.settings.update",
	"sso.teammates.create",
	"sso.teammates.update",
	"stats.global.read",
	"stats.read",
	"subusers.create",
	"subusers.credits.create",
	"subusers.credits.delete",
	"subusers.credits.read",
	"subusers.credits.remaining.create",
	"subusers.credits.remaining.delete",
	"subusers.credits.remaining.read",
	"subusers.credits.remaining.update",
	"subusers.credits.update",
	"subusers.delete",
	"subusers.monitor.create",
	"subusers.monitor.delete",
	"subusers.monitor.read",
	"subusers.monitor.update",
	"subusers.read",
	"subusers.reputations.read",
	"subusers.stats.monthly.read",
	"subusers.stats.read",
	"subusers.stats.sums.read",
	"subusers.summary.read",
	"subusers.update",
	"suppression.blocks.create",
	"suppression.blocks.delete",
	"suppression.blocks.read",
	"suppression.blocks.update",
	"suppression.bounces.create",
	"suppression.bounces.delete",
	"suppression.bounces.read",
	"suppression.bounces.update",
	"suppression.create",
	"suppression.delete",
	"suppression.invalid_emails.create",
	"suppression.invalid_emails.delete",
	"suppression.invalid_emails.read",
	"suppression.invalid_emails.update",
	"suppression.read",
	"suppression.spam_reports.create",
	"suppression.spam_reports.delete",
	"suppression.spam_reports.read",
	"suppression.spam_reports.update",
	"suppression.unsubscribes.create",
	"suppression.unsubscribes.delete",
	"suppression.unsubscribes.read",
	"suppression.unsubscribes.update",
	"suppression.update",
	"teammates.create",
	"teammates.delete",
	"teammates.read",
	"teammates.update",
	"templates.create",
	"templates.delete",
	"templates.read",
	"templates.update",
	"templates.versions.activate.create",
	"templates.versions.activate.delete",
	"templates.versions.activate.read",
	"templates.versions.activate.update",
	"templates.versions.create",
	"templates.versions.delete",
	"templates.versions.read",
	"templates.versions.update",
	"tracking_settings.click.create",
	"tracking_settings.click.delete",
	"tracking_settings.click.read",
	"tracking_settings.click.update",
	"tracking_settings.google_analytics.create",
	"tracking_settings.google_analytics.delete",
	"tracking_settings.google_analytics.read",
	"tracking_settings.google_analytics.update",
	"tracking_settings.open.create",
	"tracking_settings.open.delete",
	"tracking_settings.open.read",
	"tracking_settings.open.update",
	"tracking_settings.read",
	"tracking_settings.subscription.create",
	"tracking_settings.subscription.delete",
	"tracking_settings.subscription.read",
	"tracking_settings.subscription.update",
	"ui.confirm_email",
	"ui.provision",
	"ui.signup_complete",
	"user.account.read",
	"user.credits.read",
	"user.email.read",
	"user.profile.create",
	"user.profile.delete",
	"user.profile.read",
	"user.profile.update",
	"user.scheduled_sends.create",
	"user.scheduled_sends.delete",
	"user.scheduled_sends.read",
	"user.scheduled_sends.update",
	"user.settings.enforced_tls.read",
	"user.settings.enforced_tls.update",
	"user.timezone.create",
	"user.timezone.delete",
	"user.timezone.read",
	"user.timezone.update",
	"user.username.read",
	"user.webhooks.event.settings.create",
	"user.webhooks.event.settings.delete",
	"user.webhooks.event.settings.read",
	"user.webhooks.event.settings.update",
	"user.webhooks.event.test.create",
	"user.webhooks.event.test.delete",
	"user.webhooks.event.test.read",
	"user.webhooks.event.test.update",
	"user.webhooks.parse.settings.create",
	"user.webhooks.parse.settings.delete",
	"user.webhooks.parse.settings.read",
	"user.webhooks.parse.settings.update",
	"user.webhooks.parse.stats.read",
	"validations.email.create",
	"validations.email.read",
	"whitelabel.create",
	"whitelabel.delete",
	"whitelabel.read",
	"whitelabel.update",
}

func newTeammateResource() resource.Resource {
	return &teammateResource{}
}

type teammateResource struct {
	client *sendgrid.Client
}

type teammateResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Email    types.String   `tfsdk:"email"`
	IsAdmin  types.Bool     `tfsdk:"is_admin"`
	Scopes   []types.String `tfsdk:"scopes"`
	Username types.String   `tfsdk:"username"`
}

func (r *teammateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_teammate"
}

func (r *teammateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provides a Teammate resource.

Teammates is an account administration and security tool designed to help manage multiple users on a single SendGrid account. Teammates is built for groups of shared users, where each user has a different role and thus requires access to different SendGrid features.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/glossary/teammates).
		`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Teammate's email",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Teammate's username",
				Computed:            true,
			},
			"is_admin": schema.BoolAttribute{
				MarkdownDescription: "Set to true if teammate has admin privileges.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"scopes": schema.SetAttribute{
				ElementType: types.StringType,
				MarkdownDescription: `
The permissions API Key has access to.

For more detailed information, please see the [SendGrid documentation](https://docs.sendgrid.com/ui/account-and-settings/teammate-permissions#persona-scopes)

The following Scopes are set automatically by SendGrid, so they cannot be set manually:

- 2fa_exempt
- 2fa_required
- sender_verification_exempt
- sender_verification_eligible

A teammate remains in a pending state until the invitation is accepted, during which scopes cannot be modified.
`,
				Required: true,
			},
		},
	}
}

func (r *teammateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sendgrid.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sendgrid.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *teammateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data teammateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &sendgrid.InputInviteTeammate{
		Email:   data.Email.ValueString(),
		IsAdmin: data.IsAdmin.ValueBool(),
	}

	// adminitors have all scopes, so we don't need to set them.
	if !data.IsAdmin.ValueBool() {
		var scopes []string
		for _, s := range data.Scopes {
			// If scopes automatically added by SendGrid is specified, the process should fail.
			if slices.Contains(autoScopes, s.ValueString()) {
				resp.Diagnostics.AddError(
					"Creating teammate",
					fmt.Sprintf(
						"Unable to create teammate, got error: scopes automatically by SendGrid and cannot be manually assigned: %s",
						strings.Join(autoScopes, ", "),
					),
				)
				return
			}
			// If scopes are invalid, the process should fail.
			if !slices.Contains(validScopes, s.ValueString()) {
				resp.Diagnostics.AddError(
					"Creating teammate",
					fmt.Sprintf("Unable to create teammate, got error: scope '%s' is not valid", s.ValueString()),
				)
				return
			}
			scopes = append(scopes, s.ValueString())
		}

		input.Scopes = scopes
	}

	// NOTE: Re-execute after the re-executable time has elapsed when a rate limit occurs
	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.InviteTeammate(context.TODO(), input)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating teammate",
			fmt.Sprintf("Unable to invite teammate, got error: %s", err),
		)
		return
	}

	inviteTeammate, ok := res.(*sendgrid.OutputInviteTeammate)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating teammate",
			"Failed to assert type *sendgrid.OutputInviteTeammate",
		)
		return
	}

	scopes := []types.String{}
	if !inviteTeammate.IsAdmin {
		for _, s := range inviteTeammate.Scopes {
			if slices.Contains(autoScopes, s) {
				continue
			}
			scopes = append(scopes, types.StringValue(s))
		}
	}

	// pending user does not have an username.
	data = teammateResourceModel{
		ID:      types.StringValue(inviteTeammate.Email),
		Email:   types.StringValue(inviteTeammate.Email),
		IsAdmin: types.BoolValue(inviteTeammate.IsAdmin),
		Scopes:  scopes,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *teammateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data teammateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	email := data.Email.ValueString()

	pendingTeammate, err := pendingTeammateByEmail(ctx, r.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading teammate",
			fmt.Sprintf("Unable to get pending teammates, got error: %s", err),
		)
		return
	}

	// If the teammate is in a pending state, return their data.
	if pendingTeammate != nil {
		scopes := []types.String{}
		// administorators have all scopes, so we don't need to set them.
		if !data.IsAdmin.ValueBool() {
			for _, s := range pendingTeammate.Scopes {
				if slices.Contains(autoScopes, s) {
					continue
				}
				scopes = append(scopes, types.StringValue(s))
			}
		}
		data = teammateResourceModel{
			ID:    types.StringValue(pendingTeammate.Email),
			Email: types.StringValue(pendingTeammate.Email),
			// NOTE: As per the SendGrid API specifications,
			//       pending teammates cannot update the administrator flag.
			//       In such cases, discrepancies arise between the Terraform code and the tfstate,
			//       leading to errors during the execution of terraform apply.
			//       For pending teammates, it update the is_admin value in the tfstate to prevent any discrepancies.
			//       While there might be differences from the actual code,
			//       not accommodating the above would hinder team member management, making it unavoidable.
			IsAdmin: data.IsAdmin,
			Scopes:  scopes,
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	teammateByEmail, err := getTeammateByEmail(ctx, r.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading teammate",
			fmt.Sprintf("Unable to read teammate (%s), got error: %s", email, err),
		)
		return
	}

	// If you are unable to retrieve your teammate's information using their email address,
	// it removes the resource information from the state.
	if teammateByEmail == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	o, err := r.client.GetTeammate(ctx, teammateByEmail.Username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading teammate",
			fmt.Sprintf("Unable to read teammate (username: %s), got error: %s", teammateByEmail.Username, err),
		)
		return
	}

	scopes := []types.String{}
	// admin users have all scopes, so we don't need to set them.
	if !o.IsAdmin {
		for _, s := range o.Scopes {
			// Automatically assigned scopes in SendGrid are not managed.
			if slices.Contains(autoScopes, s) {
				continue
			}
			scopes = append(scopes, types.StringValue(s))
		}
	}

	data = teammateResourceModel{
		ID:       types.StringValue(o.Email),
		Email:    types.StringValue(o.Email),
		IsAdmin:  types.BoolValue(o.IsAdmin),
		Username: types.StringValue(o.Username),
		Scopes:   scopes,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *teammateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state teammateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	email := data.Email.ValueString()

	pendingTeammate, err := pendingTeammateByEmail(ctx, r.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating teammate",
			fmt.Sprintf("Unable to get pending teammates, got error: %s", err),
		)
		return
	}

	// If the teammate is in a pending state, it is not possible to update the permissions.
	if pendingTeammate != nil {
		scopes := []types.String{}
		if !data.IsAdmin.ValueBool() {
			scopes = data.Scopes
		}
		p := teammateResourceModel{
			ID:    types.StringValue(pendingTeammate.Email),
			Email: types.StringValue(pendingTeammate.Email),
			// NOTE: As per the SendGrid API specifications,
			//       pending teammates cannot update the administrator flag and scopes.
			//       In such cases, discrepancies arise between the Terraform code and the tfstate,
			//       leading to errors during the execution of terraform apply.
			//       For pending teammates, it update the is_admin value in the tfstate to prevent any discrepancies.
			//       While there might be differences from the actual code,
			//       not accommodating the above would hinder team member management, making it unavoidable.
			IsAdmin: data.IsAdmin,
			Scopes:  scopes,
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
		return
	}

	// get username from tfstate
	username := state.Username.ValueString()

	scopes := []string{}
	if !data.IsAdmin.ValueBool() {
		for _, s := range data.Scopes {
			// If scopes automatically added by SendGrid is specified, the process should fail.
			if slices.Contains(autoScopes, s.ValueString()) {
				resp.Diagnostics.AddError(
					"Updating teammate",
					fmt.Sprintf(
						"Unable to update teammate, got error: scopes automatically by SendGrid and cannot be manually assigned: %s",
						strings.Join(autoScopes, ", "),
					),
				)
				return
			}
			// If scopes are invalid, the process should fail.
			if !slices.Contains(validScopes, s.ValueString()) {
				resp.Diagnostics.AddError(
					"Updating teammate",
					fmt.Sprintf("Unable to update teammate, got error: scope '%s' is not valid", s.ValueString()),
				)
				return
			}

			scopes = append(scopes, s.ValueString())
		}
	}

	o, err := r.client.UpdateTeammatePermissions(ctx, username, &sendgrid.InputUpdateTeammatePermissions{
		IsAdmin: data.IsAdmin.ValueBool(),
		Scopes:  scopes,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Updating teammate",
			fmt.Sprintf("Unable to update teammate permissions, got error: %s", err),
		)
		return
	}

	scopesSet := []types.String{}
	if !o.IsAdmin {
		for _, s := range o.Scopes {
			scopesSet = append(scopesSet, types.StringValue(s))
		}
	}

	// Save updated data into Terraform state
	data = teammateResourceModel{
		ID:       types.StringValue(o.Email),
		Email:    types.StringValue(o.Email),
		IsAdmin:  types.BoolValue(o.IsAdmin),
		Username: types.StringValue(o.Username),
		Scopes:   scopesSet,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *teammateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data teammateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	email := data.Email.ValueString()

	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		// Invited users are treated as pending users until they set up their profiles.
		return pendingTeammateByEmail(ctx, r.client, email)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting teammate",
			fmt.Sprintf("Unable to get pending teammates, got error: %s", err),
		)
		return
	}

	pendingUser, ok := res.(*sendgrid.PendingTeammate)
	if !ok {
		resp.Diagnostics.AddError(
			"Deleting teammate",
			"Failed to assert type *sendgrid.PendingTeammate",
		)
		return
	}

	if pendingUser != nil {
		_, err = retryOnRateLimit(ctx, func() (interface{}, error) {
			return nil, r.client.DeletePendingTeammate(ctx, pendingUser.Token)
		})
		// If the teammate is in a pending state, execute the API to remove pending teammates.
		if err != nil {
			resp.Diagnostics.AddError(
				"Deleting teammate",
				fmt.Sprintf("Unable to delete pending teammate, got error: %s", err),
			)
		}
		return
	}

	res, err = retryOnRateLimit(ctx, func() (interface{}, error) {
		return getTeammateByEmail(ctx, r.client, email)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting teammate",
			fmt.Sprintf("Unable to get teammates, got error: %s", err),
		)
		return
	}

	teammateByEmail, ok := res.(*sendgrid.Teammate)
	if !ok {
		resp.Diagnostics.AddError(
			"Deleting teammate",
			"Failed to assert type *sendgrid.Teammate",
		)
		return
	}

	if teammateByEmail == nil {
		resp.Diagnostics.AddError(
			"Deleting teammate",
			fmt.Sprintf("Not found teammate (%s)", email),
		)
		return
	}

	_, err = retryOnRateLimit(ctx, func() (interface{}, error) {
		return nil, r.client.DeleteTeammate(ctx, teammateByEmail.Username)
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting teammate",
			fmt.Sprintf(
				"Could not delete teammate %s, unexpected error: %s",
				teammateByEmail.Username,
				err,
			),
		)
		return
	}
}

func (r *teammateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data teammateResourceModel

	email := req.ID

	resource.ImportStatePassthroughID(ctx, path.Root("email"), req, resp)

	pendingTeammate, err := pendingTeammateByEmail(ctx, r.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing teammate",
			fmt.Sprintf("Unable to get pending teammates, got error: %s", err),
		)
		return
	}

	// If the teammate is in a pending state, return their data.
	if pendingTeammate != nil {
		scopes := []types.String{}
		if !pendingTeammate.IsAdmin {
			for _, s := range pendingTeammate.Scopes {
				if slices.Contains(autoScopes, s) {
					continue
				}
				scopes = append(scopes, types.StringValue(s))
			}
		}
		data = teammateResourceModel{
			ID:      types.StringValue(email),
			Email:   types.StringValue(email),
			IsAdmin: types.BoolValue(pendingTeammate.IsAdmin),
			Scopes:  scopes,
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	teammateByEmail, err := getTeammateByEmail(ctx, r.client, email)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing teammate",
			fmt.Sprintf("Unable to read teammate (%s), got error: %s", email, err),
		)
		return
	}

	if teammateByEmail == nil {
		resp.Diagnostics.AddError(
			"Importing teammate",
			fmt.Sprintf("Not found teammate (%s)", email),
		)
		return
	}

	teammate, err := r.client.GetTeammate(ctx, teammateByEmail.Username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing teammate",
			fmt.Sprintf("Unable to read teammate, got error: %s", err),
		)
		return
	}

	scopes := []types.String{}
	if !teammate.IsAdmin {
		for _, s := range teammate.Scopes {
			// Automatically assigned scopes in SendGrid are not managed.
			if slices.Contains(autoScopes, s) {
				continue
			}
			scopes = append(scopes, types.StringValue(s))
		}
	}

	data = teammateResourceModel{
		ID:       types.StringValue(teammate.Email),
		Email:    types.StringValue(teammate.Email),
		IsAdmin:  types.BoolValue(teammate.IsAdmin),
		Username: types.StringValue(teammate.Username),
		Scopes:   scopes,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
