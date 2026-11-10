variable "oauth_client_secret" {
  type      = string
  sensitive = true
}

resource "sendgrid_webhook_security_policy" "example" {
  name              = "inbound-parse"
  signature_enabled = true

  oauth_client_id     = "client-id"
  oauth_client_secret = var.oauth_client_secret
  oauth_token_url     = "https://example.com/oauth/token"
  oauth_scopes        = ["webhooks:write"]

  lifecycle {
    create_before_destroy = true
  }
}

resource "sendgrid_inbound_parse_webhook" "example" {
  hostname        = "parse.example.com"
  url             = "https://example.com/inbound"
  security_policy = sendgrid_webhook_security_policy.example.id
}
