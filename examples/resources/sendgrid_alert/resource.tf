resource "sendgrid_alert" "example" {
  type       = "usage_limit"
  email_to   = "dummy@example.com"
  percentage = 90
}
