resource "sendgrid_subscription_tracking_settings" "example" {
  enabled       = true
  html_content  = "<p>If you would like to unsubscribe, <% click here %>.</p>"
  plain_content = "If you would like to unsubscribe, click here: <% %>."
  url           = "https://example.com/unsubscribe"
}
