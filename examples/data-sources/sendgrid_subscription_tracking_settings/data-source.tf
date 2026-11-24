data "sendgrid_subscription_tracking_settings" "example" {}

output "enabled" {
  value = data.sendgrid_subscription_tracking_settings.example.enabled
}
