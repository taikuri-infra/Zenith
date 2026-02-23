output "server_ip" {
  description = "Staging server IP"
  value       = var.create_server ? module.staging_server[0].server_ip : var.existing_server_ip
}

output "dns_records" {
  description = "Created DNS records"
  value       = module.dns.platform_hostnames
}

output "ansible_inventory_hint" {
  description = "Use this IP in infra/ansible/inventory/staging.yml"
  value       = "ansible_host: ${var.create_server ? module.staging_server[0].server_ip : var.existing_server_ip}"
}
