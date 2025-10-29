resource "sendgrid_subuser" "example" {
  username            = "dummy"
  email               = "dummy@example.com"
  password_wo         = "dummydummy1!"
  password_wo_version = 1
  ips                 = ["1.1.1.2"]
}
