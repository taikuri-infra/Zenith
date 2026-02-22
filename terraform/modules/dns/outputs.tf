output "platform_hostnames" {
  description = "Created platform DNS hostnames"
  value       = { for k, r in cloudflare_record.platform : k => r.hostname }
}

output "customer_hostnames" {
  description = "Created customer DNS hostnames"
  value       = { for k, r in cloudflare_record.customer : k => r.hostname }
}
