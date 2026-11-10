variable "oauth_client_secret" {
  type      = string
  sensitive = true
}

resource "sendgrid_webhook_security_policy" "example" {
  name = "inbound-parse"

  signature = {
    enabled = true
  }

  oauth = {
    client_id     = "client-id"
    client_secret = var.oauth_client_secret
    token_url     = "https://example.com/oauth/token"
    scopes        = ["webhooks:write"]
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "sendgrid_inbound_parse_webhook" "example" {
  hostname        = "parse.example.com"
  url             = "https://example.com/inbound"
  security_policy = sendgrid_webhook_security_policy.example.id
}
