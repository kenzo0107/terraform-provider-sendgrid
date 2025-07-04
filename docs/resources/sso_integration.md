---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "sendgrid_sso_integration Resource - terraform-provider-sendgrid"
subcategory: ""
description: |-
  Provides SSO Integration resource.
---

# sendgrid_sso_integration (Resource)

Provides SSO Integration resource.

## Example Usage

```terraform
resource "sendgrid_sso_integration" "example" {
  name    = "idp"
  enabled = false

  signin_url  = "https://example.com/signin"
  signout_url = "https://example.com/signout"
  entity_id   = "https://example.com/1234567"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `enabled` (Boolean) Indicates if the integration is enabled.
- `entity_id` (String) An identifier provided by your IdP to identify Twilio SendGrid in the SAML interaction. This is called the "SAML Issuer ID" in the Twilio SendGrid UI.
- `name` (String) The name of your integration. This name can be anything that makes sense for your organization (eg. Twilio SendGrid)
- `signin_url` (String) The IdP's SAML POST endpoint. This endpoint should receive requests and initiate an SSO login flow. This is called the "Embed Link" in the Twilio SendGrid UI.
- `signout_url` (String) This URL is relevant only for an IdP-initiated authentication flow. If a user authenticates from their IdP, this URL will return them to their IdP when logging out.

### Read-Only

- `audience_url` (String) The URL where your IdP should POST its SAML response. This is the Twilio SendGrid URL that is responsible for receiving and parsing a SAML assertion. This is the same URL as the Single Sign-On URL when using SendGrid.
- `completed_integration` (Boolean) Indicates if the integration is complete.
- `id` (String) A unique ID assigned to the configuration by SendGrid.
- `single_signon_url` (String) The URL where your IdP should POST its SAML response. This is the Twilio SendGrid URL that is responsible for receiving and parsing a SAML assertion. This is the same URL as the Audience URL when using SendGrid.

## Import

Import is supported using the following syntax:

The [`terraform import` command](https://developer.hashicorp.com/terraform/cli/commands/import) can be used, for example:

```shell
% terraform import sendgrid_sso_integration.example <integration id>
```
