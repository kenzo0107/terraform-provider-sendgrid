resource "sendgrid_api_key" "example" {
  name = "dummy"
  scopes = [
    "user.profile.read",
  ]
}
