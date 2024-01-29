
data "sendgrid_template_version" "example" {
  template_id = "d-1234567890abcdefghijklmnopqrstuv"
  id          = "abcde123-fg45-6789-012e-3456789abcde"
}

output "name" {
  value = data.sendgrid_template_version.example.name
}
