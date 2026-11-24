data "sendgrid_google_analytics_settings" "example" {}

output "utm_source" {
  value = data.sendgrid_google_analytics_settings.example.utm_source
}
