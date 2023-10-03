resource "sendgrid_sender_authentication" "example" {
  domain = "example.com"
}

output "ips" {
  value = sendgrid_sender_authentication.example.ips
}
