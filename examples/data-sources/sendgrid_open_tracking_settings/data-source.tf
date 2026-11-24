data "sendgrid_open_tracking_settings" "example" {}

output "enabled" {
  value = data.sendgrid_open_tracking_settings.example.enabled
}
