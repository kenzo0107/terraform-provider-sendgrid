data "sendgrid_design" "example" {
  id = "your-design-id"
}

output "design_name" {
  value = data.sendgrid_design.example.name
}
