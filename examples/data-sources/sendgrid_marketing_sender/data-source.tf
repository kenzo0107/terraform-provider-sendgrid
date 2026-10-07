data "sendgrid_marketing_sender" "example" {
  id = "12345678"
}

output "from_email" {
  value = data.sendgrid_marketing_sender.example.from_email
}
