resource "sendgrid_teammate" "example" {
  email = "dummy@example.com"
  scopes = [
    "user.profile.read",
  ]

  lifecycle {
    ignore_changes = [scopes]
  }
}
