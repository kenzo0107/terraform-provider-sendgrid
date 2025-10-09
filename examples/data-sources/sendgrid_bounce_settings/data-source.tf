data "sendgrid_bounce_settings" "example" {}

output "soft_bounce_purge_days" {
  value = data.sendgrid_bounce_settings.example.soft_bounce_purge_days
}