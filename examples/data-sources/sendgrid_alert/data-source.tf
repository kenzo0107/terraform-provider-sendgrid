data "sendgrid_alert" "example" {
  id = "1234567"
}

output "email_to" {
  value = data.sendgrid_alert.example.email_to
}

output "type" {
  value = data.sendgrid_alert.example.type
}

output "frequency" {
  value = data.sendgrid_alert.example.frequency
}
