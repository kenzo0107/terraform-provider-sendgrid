resource "sendgrid_sso_integration" "example" {
  name    = "idp"
  enabled = false

  signin_url  = "https://example.com/signin"
  signout_url = "https://example.com/signout"
  entity_id   = "https://example.com/1234567"
}
