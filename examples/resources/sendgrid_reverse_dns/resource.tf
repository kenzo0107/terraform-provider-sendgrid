resource "sendgrid_reverse_dns" "example" {
  ip        = "127.0.0.1"
  domain    = "example.com"
  subdomain = "dummy"
}
