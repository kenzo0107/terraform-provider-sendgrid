data "sendgrid_api_key" "example" {
  id = "dummy"
}

output "name" {
  value = data.sendgrid_api_key.example.name
}

output "scopes" {
  value = data.sendgrid_api_key.example.scopes
}
