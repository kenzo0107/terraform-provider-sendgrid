
data "sendgrid_template" "example" {
  id = "d-1234567890abcdefghijklmnopqrstuv"
}

output "name" {
  value = data.sendgrid_template.example.name
}
