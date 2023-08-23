resource "sendgrid_subuser" "example" {
  username = "dummy"
  email    = "dummy@example.com"
  password = "dummy"
  ips      = ["1.1.1.2"]
}
