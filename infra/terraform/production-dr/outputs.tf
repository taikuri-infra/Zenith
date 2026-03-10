output "dr_server_ip" {
  description = "Public IP of the DR server in Helsinki"
  value       = hcloud_server.dr.ipv4_address
}

output "dr_server_id" {
  description = "Hetzner server ID"
  value       = hcloud_server.dr.id
}

output "dr_server_status" {
  description = "Server status"
  value       = hcloud_server.dr.status
}

output "dr_location" {
  description = "Datacenter location"
  value       = hcloud_server.dr.location
}

output "kubeconfig_command" {
  description = "Command to fetch kubeconfig from DR server"
  value       = "scp root@${hcloud_server.dr.ipv4_address}:/etc/rancher/k3s/k3s.yaml ~/.kube/zenith-dr.yaml && sed -i '' 's/127.0.0.1/${hcloud_server.dr.ipv4_address}/' ~/.kube/zenith-dr.yaml"
}
