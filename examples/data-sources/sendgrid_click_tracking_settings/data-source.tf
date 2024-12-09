data "sendgrid_click_tracking_settings" "example" {}

output "enable_text" {
  value = data.sendgrid_click_tracking_settings.example.enable_text
}
