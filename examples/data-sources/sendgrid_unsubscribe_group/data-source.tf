data "sendgrid_unsubscribe_group" "example" {
  id = "13253"
}

output "name" {
  value = data.sendgrid_unsubscribe_group.example.name
}
