data "sendgrid_sender_verification" "example" {
  id = "12345678"
}

output "nickname" {
  value = data.sendgrid_sender_verification.example.nickname
}
