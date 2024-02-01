resource "sendgrid_enforce_tls" "example" {
  version            = 1.2
  require_tls        = false
  require_valid_cert = false
}
