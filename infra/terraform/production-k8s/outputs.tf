output "cert_manager" {
  description = "cert-manager status"
  value       = module.platform.cert_manager_status
}

output "zenith" {
  description = "Zenith platform status"
  value       = module.platform.zenith_status
}

output "zenith_version" {
  description = "Deployed Zenith chart version"
  value       = module.platform.zenith_version
}

output "keda" {
  description = "KEDA status"
  value       = module.platform.keda_status
}

output "monitoring" {
  description = "Monitoring status"
  value       = module.platform.monitoring_status
}
