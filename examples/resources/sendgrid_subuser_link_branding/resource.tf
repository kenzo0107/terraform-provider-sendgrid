resource "sendgrid_link_branding" "example" {
  domain    = "example.com"
  subdomain = "abc"
}

resource "sendgrid_subuser" "example" {
  username = "example-subuser"
  email    = "subuser@example.com"
  password = "SecurePassword123!"
  ips      = ["192.0.2.1"]
}

resource "sendgrid_subuser_link_branding" "example" {
  link_id = sendgrid_link_branding.example.id
  subuser = sendgrid_subuser.example.username
}
