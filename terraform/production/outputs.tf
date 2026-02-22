output "management_ip" {
  description = "Management plane server IP"
  value       = module.management.server_ip
}

output "cluster_ip" {
  description = "Customer cluster server IP"
  value       = module.cluster.server_ip
}

output "management_server_id" {
  description = "Management server Hetzner ID"
  value       = module.management.server_id
}

output "cluster_server_id" {
  description = "Cluster server Hetzner ID"
  value       = module.cluster.server_id
}

output "platform_dns" {
  description = "Platform DNS records"
  value       = module.platform_dns.platform_hostnames
}

output "customer_dns" {
  description = "Customer DNS records"
  value       = module.customer_dns.customer_hostnames
}

output "s3_bucket" {
  description = "S3 bucket name"
  value       = module.storage.bucket_name
}

output "ansible_hint" {
  description = "Use these IPs in ansible/inventory/production.yml"
  value       = <<-EOT
    Management: ${module.management.server_ip}
    Cluster:    ${module.cluster.server_ip}
  EOT
}
