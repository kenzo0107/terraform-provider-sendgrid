data "sendgrid_reverse_dns" "example" {
  id = "123456"
}

output "domain" {
  value = data.sendgrid_reverse_dns.example.domain
}

output "subdomain" {
  value = data.sendgrid_reverse_dns.example.subdomain
}

output "users" {
  value = data.sendgrid_reverse_dns.example.users
}
