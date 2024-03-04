data "sendgrid_enforce_tls" "example" {}

output "version" {
  value = data.sendgrid_enforce_tls.example.version
}
