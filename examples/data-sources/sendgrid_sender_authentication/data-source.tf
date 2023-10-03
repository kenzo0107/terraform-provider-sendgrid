data "sendgrid_sender_authentication" "example" {
  id = "123456789"
}

output "domain" {
  value = data.sendgrid_sender_authentication.example.domain
}
