output "server_ip" {
  description = "Public IPv4 address of the server"
  value       = hcloud_server.zenith.ipv4_address
}

output "server_id" {
  description = "Hetzner server ID"
  value       = hcloud_server.zenith.id
}

output "server_name" {
  description = "Server name"
  value       = hcloud_server.zenith.name
}

output "server_status" {
  description = "Server status"
  value       = hcloud_server.zenith.status
}
