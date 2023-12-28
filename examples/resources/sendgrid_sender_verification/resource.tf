resource "sendgrid_sender_verification" "example" {
  reply_to   = "noreply@example.com"
  from_name  = "dummy"
  from_email = "dummy@example.com"
  address    = "dummy"
  address2   = "dummy"
  city       = "dummy"
  country    = "JPN"
  nickname   = "dummy"
}
