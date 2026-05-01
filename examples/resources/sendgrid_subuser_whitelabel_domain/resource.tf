resource "sendgrid_sender_authentication" "example" {
  domain = "example.com"
}

resource "sendgrid_subuser" "example" {
  username = "example-subuser"
  email    = "subuser@example.com"
  password = "SecurePassword123!"
  ips      = ["192.0.2.1"]
}

resource "sendgrid_subuser_whitelabel_domain" "example" {
  domain_id = sendgrid_sender_authentication.example.id
  subuser   = sendgrid_subuser.example.username
}
