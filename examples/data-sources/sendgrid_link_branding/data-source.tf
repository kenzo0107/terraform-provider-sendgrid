data "sendgrid_link_branding" "example" {
  id = "3008665"
}

output "domain" {
  value = data.sendgrid_link_branding.example.domain
}

output "dns" {
  value = data.sendgrid_link_branding.example.dns
}
