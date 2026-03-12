data "sendgrid_ip_pool" "example" {
  name = "marketing"
}

output "ip_pool_name" {
  value = data.sendgrid_ip_pool.example.name
}

output "ip_pool_ips" {
  value = data.sendgrid_ip_pool.example.ips
}
