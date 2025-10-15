data "sendgrid_bounce_settings" "example" {}

output "bounce_settings_enabled" {
  value = data.sendgrid_bounce_settings.example.enabled
}
output "soft_bounce_purge_days" {
  value = data.sendgrid_bounce_settings.example.soft_bounces
}
output "hard_bounce_purge_days" {
  value = data.sendgrid_bounce_settings.example.hard_bounces
}
