data "sendgrid_subuser" "example" {
  username = "dummy"
}

output "email" {
  value = data.sendgrid_subuser.example.email
}

output "user_id" {
  value = data.sendgrid_subuser.example.id
}
