resource "sendgrid_inbound_parse_webhook" "example" {
  hostname   = "example.com"
  url        = "https://foo.bar"
  spam_check = true
  send_raw   = true
}
