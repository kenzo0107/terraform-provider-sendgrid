resource "sendgrid_ip_access_management_allow_list" "example" {
  ips = [
    "192.0.2.10",
    "198.51.100.0/24",
    "203.0.*.*",
  ]

  acknowledge_lockout_risk = true
}
