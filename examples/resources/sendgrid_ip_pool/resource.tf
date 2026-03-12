resource "sendgrid_ip_pool" "example" {
  name = "marketing"
  ips = [
    "111.11.111.111",
  ]
}
