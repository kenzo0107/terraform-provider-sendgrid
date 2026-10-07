resource "sendgrid_marketing_sender" "example" {
  nickname      = "example"
  from_email    = "marketing@example.com"
  from_name     = "Example Inc."
  reply_to      = "support@example.com"
  reply_to_name = "Example Support"
  address       = "1-1-1 Chiyoda"
  city          = "Chiyoda-ku"
  state         = "TK"
  zip           = "100-0001"
  country       = "Japan"
}
