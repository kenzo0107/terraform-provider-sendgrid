resource "sendgrid_event_webhook" "example" {
  url           = "https://example.com"
  enabled       = true
  signed        = true
  delivered     = true
  processed     = true
  bounce        = true
  dropped       = true
  friendly_name = "Example Event Webhook"
}
