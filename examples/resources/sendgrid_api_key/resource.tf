resource "sendgrid_api_key" "example" {
  name = "dummy"
  scopes = [
    "user.profile.read",
    "sender_verification_exempt",
    "2fa_required",
  ]
}
