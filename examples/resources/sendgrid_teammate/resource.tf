resource "sendgrid_teammate" "example" {
  email = "dummy@example.com"
  scopes = [
    "user.profile.read",
    "mail_settings.read",
    "partner_settings.read",
    "tracking_settings.read",
    "user.account.read",
    "user.credits.read",
    "user.email.read",
    "user.profile.update",
    "user.settings.enforced_tls.read",
    "user.timezone.read",
    "user.username.read",
  ]
}
