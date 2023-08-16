data "sendgrid_teammate" "example" {
  email = "dummy@example.com"
}

output "is_admin" {
  value = data.sendgrid_teammate.example.is_admin
}

output "scopes" {
  value = data.sendgrid_teammate.example.scopes
}
