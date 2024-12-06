resource "sendgrid_sso_teammate" "example" {
  email      = "dummy@example.com"
  first_name = "first"
  last_name  = "last"
  is_admin   = true
  scopes     = []
}
