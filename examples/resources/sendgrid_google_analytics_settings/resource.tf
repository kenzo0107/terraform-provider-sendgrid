resource "sendgrid_google_analytics_settings" "example" {
  enabled      = true
  utm_source   = "sendgrid"
  utm_medium   = "email"
  utm_campaign = "newsletter"
}
